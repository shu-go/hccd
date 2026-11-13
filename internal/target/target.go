// Package target decides which directory (and file) hccd works on, and
// builds the shell command that runs a file.
package target

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Target struct {
	Dir  string // directory the tab is for
	File string // file to run in that tab ("" if none)
}

// Resolve turns a command-line argument (may be "") into a Target. cwd is
// used when arg is empty or relative.
func Resolve(arg, cwd string) (Target, error) {
	if arg == "" {
		arg = cwd
	}
	if !filepath.IsAbs(arg) {
		arg = filepath.Join(cwd, arg)
	}
	p := filepath.Clean(arg)
	fi, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return Target{}, fmt.Errorf("not found: %s", p)
		}
		return Target{}, err
	}
	if fi.IsDir() {
		return Target{Dir: p}, nil
	}
	return Target{Dir: filepath.Dir(p), File: p}, nil
}

// SamePath compares paths the way the OS does (case-insensitive on
// Windows and macOS), ignoring trailing separators.
func SamePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return normalize(a, runtime.GOOS) == normalize(b, runtime.GOOS)
}

func normalize(p, goos string) string {
	isDrive := func(s string) bool { return goos == "windows" && len(s) == 2 && s[1] == ':' }
	if isDrive(p) {
		p += `\` // filepath.Clean("C:") is "C:."
	}
	p = filepath.Clean(p)
	if len(p) > 1 {
		p = strings.TrimRight(p, `\/`)
		if isDrive(p) {
			p += `\` // "C:" -> "C:\"
		}
	}
	if goos == "windows" || goos == "darwin" {
		p = strings.ToLower(p)
	}
	return p
}

// RunCommand is the line to type into shell (a process name such as
// "pwsh.exe", "bash", "fish") to run file.
func RunCommand(shell, file string) string {
	name := strings.ToLower(filepath.Base(shell))
	name = strings.TrimSuffix(name, ".exe")
	name = strings.TrimPrefix(name, "-") // login shells: "-zsh"
	switch name {
	case "pwsh", "powershell":
		return "& '" + strings.ReplaceAll(file, "'", "''") + "'"
	case "cmd":
		return `"` + file + `"`
	case "fish":
		r := strings.NewReplacer(`\`, `\\`, `'`, `\'`)
		return "'" + r.Replace(file) + "'"
	case "nu":
		return "^'" + file + "'"
	default: // sh, bash, zsh, dash, ksh ...
		return "'" + strings.ReplaceAll(file, "'", `'\''`) + "'"
	}
}

// ErrVirtual is returned for Explorer locations that are not file system
// folders (This PC, Home, ...).
var ErrVirtual = fmt.Errorf("not a file system folder")
