//go:build !windows

package sys

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func noConsole(cmd *exec.Cmd) {}

// Children returns the names of pid's direct children.
func Children(pid int) ([]string, error) {
	if runtime.GOOS == "linux" {
		return linuxChildren(pid)
	}
	// macOS and other BSDs
	out, err := exec.Command("pgrep", "-P", strconv.Itoa(pid)).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, nil // no children
		}
		return nil, err
	}
	var names []string
	for _, f := range strings.Fields(string(out)) {
		names = append(names, f)
	}
	return names, nil
}

func linuxChildren(pid int) ([]string, error) {
	// /proc/<pid>/task/*/children lists child pids per thread.
	tasks, err := filepath.Glob(fmt.Sprintf("/proc/%d/task/*/children", pid))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, t := range tasks {
		b, err := os.ReadFile(t)
		if err != nil {
			continue
		}
		for _, c := range strings.Fields(string(b)) {
			comm, _ := os.ReadFile("/proc/" + c + "/comm")
			names = append(names, string(bytes.TrimSpace(comm)))
		}
	}
	return names, nil
}

// ParentName returns the executable name of the parent process.
func ParentName() string {
	ppid := os.Getppid()
	if runtime.GOOS == "linux" {
		b, _ := os.ReadFile(fmt.Sprintf("/proc/%d/comm", ppid))
		return string(bytes.TrimSpace(b))
	}
	out, _ := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(ppid)).Output()
	return filepath.Base(strings.TrimSpace(string(out)))
}

func FixEnv() {}

func CanPrint() bool { return true }

func Alert(title, msg string, isErr bool) { fmt.Fprintln(os.Stderr, msg) }
