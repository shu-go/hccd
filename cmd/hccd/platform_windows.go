package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shu-go/hccd/internal/sys"
	"github.com/shu-go/hccd/internal/winx"
)

func foregroundExplorerFolder() (string, bool, error) {
	dir, ok, err := winx.ForegroundExplorerFolder()
	if ok && err == nil && (dir == "" || strings.HasPrefix(dir, "::")) {
		return "", true, fmt.Errorf("the Explorer location is not a folder on disk")
	}
	return dir, ok, err
}

const menuLabel = "Open in Herdr"

func registerMenu(register, space bool) error {
	if !register {
		if err := winx.Unregister(); err != nil {
			return err
		}
		report("Removed from the File Explorer context menu.")
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// Prefer the GUI build next to us: no console window from Explorer.
	if w := filepath.Join(filepath.Dir(exe), "hccdw.exe"); fileExists(w) {
		exe = w
	}
	var extra []string
	if space {
		extra = append(extra, "--space")
	}
	if err := winx.Register(exe, menuLabel, extra); err != nil {
		return err
	}
	report(fmt.Sprintf("Added %q to the File Explorer context menu (%s).", menuLabel, exe))
	return nil
}

func report(msg string) {
	if quietUI() {
		sys.Alert("hccd", msg, false)
	} else {
		fmt.Println(msg)
	}
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}
