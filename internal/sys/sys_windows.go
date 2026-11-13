package sys

import (
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const createNoWindow = 0x08000000

// noConsole sets CREATE_NO_WINDOW. Never set HideWindow: it passes SW_HIDE
// to GUI programs (herdr-app), whose window then stays invisible.
func noConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}

func snapshot(f func(pe *syscall.ProcessEntry32) bool) error {
	snap, err := syscall.CreateToolhelp32Snapshot(syscall.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(snap)
	var pe syscall.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	for err = syscall.Process32First(snap, &pe); err == nil; err = syscall.Process32Next(snap, &pe) {
		if !f(&pe) {
			break
		}
	}
	return nil
}

// Children returns the names of pid's direct children, without console hosts.
func Children(pid int) ([]string, error) {
	var out []string
	err := snapshot(func(pe *syscall.ProcessEntry32) bool {
		if int(pe.ParentProcessID) == pid {
			name := syscall.UTF16ToString(pe.ExeFile[:])
			if name != "conhost.exe" && name != "OpenConsole.exe" {
				out = append(out, name)
			}
		}
		return true
	})
	return out, err
}

// ParentName returns the executable name of the parent process.
func ParentName() string {
	ppid := os.Getppid()
	name := ""
	snapshot(func(pe *syscall.ProcessEntry32) bool {
		if int(pe.ProcessID) == ppid {
			name = syscall.UTF16ToString(pe.ExeFile[:])
			return false
		}
		return true
	})
	return name
}

// FixEnv fills HERDR_APP_HOME from the user environment (HKCU\Environment)
// when the inherited environment lacks it (e.g. Explorer started before it
// was set). herdr-app needs it to find the window's socket.
func FixEnv() {
	if os.Getenv("HERDR_APP_HOME") != "" {
		return
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.QUERY_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	v, _, err := k.GetStringValue("HERDR_APP_HOME")
	if err == nil && v != "" {
		if x, err := registry.ExpandString(v); err == nil {
			v = x
		}
		os.Setenv("HERDR_APP_HOME", v)
	}
}

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW = user32.NewProc("MessageBoxW")
)

// CanPrint reports whether stderr goes somewhere (a console, pipe or file).
// The -H=windowsgui build started from Explorer has no std handles.
// (A console window is not required: headless/pseudo consoles have none.)
func CanPrint() bool {
	h, err := syscall.GetStdHandle(syscall.STD_ERROR_HANDLE)
	return err == nil && h != 0 && h != syscall.InvalidHandle
}

// Alert shows a message box (used when there is no console to print to).
func Alert(title, msg string, isErr bool) {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(msg)
	flags := uintptr(0x40) // MB_ICONINFORMATION
	if isErr {
		flags = 0x10 // MB_ICONERROR
	}
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), flags)
}
