// Command hccd opens (or shows) a Herdr tab for a directory, and can run a
// file in that tab's shell.
//
//	hccd [options] [<dir>|<file>]
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/shu-go/hccd/internal/finder"
	"github.com/shu-go/hccd/internal/hc"
	"github.com/shu-go/hccd/internal/herdr"
	"github.com/shu-go/hccd/internal/sys"
	"github.com/shu-go/hccd/internal/target"
)

type options struct {
	space, always, verbose bool
	name                   string
	register, unregister   bool
	version                bool
}

// version is set at build time: -ldflags "-X main.version=1.2.3".
var version = ""

func versionString() string {
	v := version
	if v == "" {
		// go install module@vX.Y.Z records the module version.
		if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			v = bi.Main.Version
		} else {
			v = "dev"
		}
	}
	return "hccd " + v
}

var verbose bool

func logf(format string, a ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, "hccd: "+format+"\n", a...)
	}
}

func main() {
	var o options
	fs := flag.NewFlagSet("hccd", flag.ContinueOnError)
	for _, n := range []string{"s", "space"} {
		fs.BoolVar(&o.space, n, false, "create a space (not a tab) when no tab is open for the directory")
	}
	for _, n := range []string{"n", "new"} {
		fs.BoolVar(&o.always, n, false, "always create a new tab, do not look for one")
	}
	fs.StringVar(&o.name, "name", "", "name of the new tab/space (default: folder name)")
	for _, n := range []string{"v", "verbose"} {
		fs.BoolVar(&o.verbose, n, false, "explain what it does")
	}
	fs.BoolVar(&o.version, "version", false, "print the version and exit")
	if runtime.GOOS == "windows" {
		fs.BoolVar(&o.register, "register", false, "add \"Herdr で開く\" to the Explorer context menu (with -s: create spaces)")
		fs.BoolVar(&o.unregister, "unregister", false, "remove it from the Explorer context menu")
	}
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: hccd [options] [<dir>|<file>]\n\n"+
			"Shows the Herdr tab whose shell is in <dir> (default: current directory,\n"+
			"or the Explorer folder when started from Explorer), or opens one.\n"+
			"With <file>, runs it in that tab's shell.\n\noptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		os.Exit(2)
	}
	verbose = o.verbose
	if o.version {
		if quietUI() {
			sys.Alert("hccd", versionString(), false)
		} else {
			fmt.Println(versionString())
		}
		return
	}
	if fs.NArg() > 1 {
		fs.Usage()
		os.Exit(2)
	}

	if err := run(o, fs.Arg(0)); err != nil {
		if quietUI() {
			sys.Alert("hccd", err.Error(), true)
		} else {
			fmt.Fprintln(os.Stderr, "hccd:", err)
		}
		os.Exit(1)
	}
}

// quietUI: there is no console the person will read (GUI build, or a
// console that Explorer opened and closes right away).
func quietUI() bool {
	return !sys.CanPrint() || sys.ParentName() == "explorer.exe"
}

func run(o options, arg string) error {
	if o.register || o.unregister {
		return registerMenu(o.register, o.space)
	}
	sys.FixEnv()

	t, err := resolveTarget(arg)
	if err != nil {
		return err
	}
	logf("target dir=%q file=%q", t.Dir, t.File)

	started, err := herdr.EnsureWindow(15 * time.Second)
	if err != nil {
		return err
	}
	if started {
		logf("started Herdr")
	}

	c, err := hc.Dial()
	if err != nil {
		return err
	}
	defer c.Close()

	terms, err := c.List()
	if err != nil {
		return err
	}
	ls, err := herdr.List()
	if err != nil {
		return err
	}

	tab, newTerm := "", ""
	if !o.always {
		tab = finder.Tab(terms, ls, t.Dir)
	}
	if tab != "" {
		logf("found tab %s", tab)
		if err := herdr.SelectTab(tab); err != nil {
			return err
		}
	} else {
		name := o.name
		if name == "" {
			name = folderName(t.Dir)
		}
		var cr *herdr.Created
		if o.space {
			logf("new space %q", name)
			cr, err = herdr.NewSpace(t.Dir, name)
		} else {
			logf("new tab %q", name)
			cr, err = herdr.NewTab(t.Dir, name, "")
		}
		if err != nil {
			return err
		}
		tab, newTerm = cr.Tab, cr.Terminal
	}

	if t.File != "" {
		if err := runFile(c, ls, t, tab, newTerm, o.name); err != nil {
			return err
		}
	}

	// Bring the window to the front. After a cold start the first launch
	// does not raise it; this second call does.
	return herdr.Raise()
}

func resolveTarget(arg string) (target.Target, error) {
	if arg == "" {
		if dir, ok, err := explorerFolder(); ok {
			if err != nil {
				return target.Target{}, err
			}
			logf("explorer folder %q", dir)
			arg = dir
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return target.Target{}, err
	}
	return target.Resolve(arg, cwd)
}

// explorerFolder: started without arguments from Explorer (double click or
// shortcut) -> the folder of the foreground Explorer tab.
func explorerFolder() (string, bool, error) {
	if sys.ParentName() != "explorer.exe" {
		return "", false, nil
	}
	return foregroundExplorerFolder()
}

func folderName(dir string) string {
	n := filepath.Base(dir)
	if n == "" || n == "." || n == string(filepath.Separator) {
		return dir // drive or file system root
	}
	return n
}

func runFile(c *hc.Client, ls *herdr.Ls, t target.Target, tab, newTerm, name string) error {
	addr := newTerm
	if addr == "" {
		terms, err := c.List()
		if err != nil {
			return err
		}
		if idle, ok := finder.IdleShell(terms, tab, t.Dir, busy); ok {
			addr = idle.Address
			logf("idle shell %s", addr)
		} else {
			if name == "" {
				name = folderName(t.Dir)
			}
			logf("no idle shell in tab %s: new tab", tab)
			cr, err := herdr.NewTab(t.Dir, name, spaceOf(ls, tab))
			if err != nil {
				return err
			}
			addr = cr.Terminal
		}
	}
	term, err := waitReady(c, addr, t.Dir)
	if err != nil {
		return err
	}
	shell := ""
	if term.Foreground != nil {
		shell = term.Foreground.Name
	}
	if shell == "" && len(term.Argv) > 0 {
		shell = term.Argv[0]
	}
	line := target.RunCommand(shell, t.File)
	logf("send to %s (%s): %s", addr, shell, line)
	return c.Send(addr, line)
}

// busy: an agent (status) is in it, or the shell runs a child process.
// Outside Windows hc's foreground follows the foreground job too.
func busy(t hc.Terminal) bool {
	if t.HasStatus() || t.Foreground == nil {
		return true
	}
	if runtime.GOOS != "windows" && t.Pid != 0 && t.Foreground.Pid != t.Pid {
		return true
	}
	kids, err := sys.Children(t.Foreground.Pid)
	return err != nil || len(kids) > 0
}

func spaceOf(ls *herdr.Ls, tab string) string {
	for _, s := range ls.Spaces {
		for _, t := range s.Tabs {
			if t.ID == tab {
				return s.ID
			}
		}
	}
	return ""
}

// waitReady waits until the new shell reports it is in dir (on Windows the
// prompt hook sets it, so the prompt is up).
func waitReady(c *hc.Client, addr, dir string) (*hc.Terminal, error) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		t, err := c.Get(addr)
		if err == nil && t.Foreground != nil && target.SamePath(t.CurrentDir(), dir) {
			time.Sleep(200 * time.Millisecond)
			return t, nil
		}
		if time.Now().After(deadline) {
			if err == nil {
				err = fmt.Errorf("shell in %s is not ready in %s", addr, dir)
			}
			return nil, err
		}
		time.Sleep(200 * time.Millisecond)
	}
}
