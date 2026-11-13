package winx

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const menuKey = "hccd"

// menuEntries: HKCU\Software\Classes\<class>\shell\hccd -> placeholder.
var menuEntries = []struct{ class, arg string }{
	{`Directory\Background`, "%V"}, // background of an open folder
	{`Directory`, "%1"},            // a folder
	{`Drive`, "%1"},                // a drive (C:\)
	{`*`, "%1"},                    // a file: run it
}

// Register adds "Open in Herdr" to the Explorer context menu for the current
// user. exe should be the GUI build (hccdw.exe). extra are flags placed
// before the path (e.g. "--space").
func Register(exe, label string, extra []string) error {
	for _, e := range menuEntries {
		base := `Software\Classes\` + e.class + `\shell\` + menuKey
		k, _, err := registry.CreateKey(registry.CURRENT_USER, base, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("create %s: %w", base, err)
		}
		k.SetStringValue("", label)
		k.SetStringValue("Icon", exe)
		k.Close()
		c, _, err := registry.CreateKey(registry.CURRENT_USER, base+`\command`, registry.SET_VALUE)
		if err != nil {
			return fmt.Errorf("create %s\\command: %w", base, err)
		}
		args := append([]string{`"` + exe + `"`}, extra...)
		args = append(args, `"`+e.arg+`"`)
		err = c.SetStringValue("", strings.Join(args, " "))
		c.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func Unregister() error {
	for _, e := range menuEntries {
		base := `Software\Classes\` + e.class + `\shell\` + menuKey
		registry.DeleteKey(registry.CURRENT_USER, base+`\command`)
		if err := registry.DeleteKey(registry.CURRENT_USER, base); err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("delete %s: %w", base, err)
		}
	}
	return nil
}
