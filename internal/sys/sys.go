// Package sys hides OS differences: starting console children without a
// window, child-process lookup, and the parent process name.
package sys

import "os/exec"

// Command returns an exec.Cmd for a console program (hc, herdr-app
// subcommands) that never opens a console window on Windows.
func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	noConsole(cmd)
	return cmd
}
