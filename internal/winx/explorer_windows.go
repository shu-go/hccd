// Package winx holds the Windows-only parts: the Explorer folder lookup and
// the Explorer context menu registration.
package winx

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetClassNameW       = user32.NewProc("GetClassNameW")
	procFindWindowExW       = user32.NewProc("FindWindowExW")

	iidServiceProvider  = ole.NewGUID("{6D5140C1-7436-11CE-8034-00AA006009FA}")
	sidSTopLevelBrowser = ole.NewGUID("{4C96BE40-915C-11CF-99D3-00AA004AE837}")
	iidShellBrowser     = ole.NewGUID("{000214E2-0000-0000-C000-000000000046}")
)

func className(h uintptr) string {
	buf := make([]uint16, 256)
	procGetClassNameW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

// activeTab returns the first ShellTabWindowClass child in Z-order: the
// tab being shown (verified on Windows 11, see verify/VERIFICATION.md).
func activeTab(top uintptr) uintptr {
	cls, _ := syscall.UTF16PtrFromString("ShellTabWindowClass")
	h, _, _ := procFindWindowExW.Call(top, 0, uintptr(unsafe.Pointer(cls)), 0)
	return h
}

func vtbl(p unsafe.Pointer) *[8]uintptr { return *(**[8]uintptr)(p) }

// tabWindow: IServiceProvider::QueryService(SID_STopLevelBrowser,
// IID_IShellBrowser) -> IOleWindow::GetWindow.
func tabWindow(item *ole.IDispatch) uintptr {
	sp, err := item.QueryInterface(iidServiceProvider)
	if err != nil {
		return 0
	}
	defer sp.Release()
	var sb unsafe.Pointer
	hr, _, _ := syscall.SyscallN(vtbl(unsafe.Pointer(sp))[3], uintptr(unsafe.Pointer(sp)),
		uintptr(unsafe.Pointer(sidSTopLevelBrowser)), uintptr(unsafe.Pointer(iidShellBrowser)), uintptr(unsafe.Pointer(&sb)))
	if hr != 0 || sb == nil {
		return 0
	}
	defer syscall.SyscallN(vtbl(sb)[2], uintptr(sb))
	var h uintptr
	if hr, _, _ := syscall.SyscallN(vtbl(sb)[3], uintptr(sb), uintptr(unsafe.Pointer(&h))); hr != 0 {
		return 0
	}
	return h
}

type explorerTab struct {
	hwnd, tab uintptr
	path      string
}

func explorerTabs() ([]explorerTab, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		if oe, ok := err.(*ole.OleError); !ok || oe.Code() != 1 { // S_FALSE is fine
			return nil, fmt.Errorf("CoInitializeEx: %w", err)
		}
	}
	defer ole.CoUninitialize()

	unk, err := oleutil.CreateObject("Shell.Application")
	if err != nil {
		return nil, fmt.Errorf("Shell.Application: %w", err)
	}
	defer unk.Release()
	shell, err := unk.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, err
	}
	defer shell.Release()
	wv, err := oleutil.CallMethod(shell, "Windows")
	if err != nil {
		return nil, err
	}
	wins := wv.ToIDispatch()
	defer wins.Release()
	cv, err := oleutil.GetProperty(wins, "Count")
	if err != nil {
		return nil, err
	}
	var out []explorerTab
	for i := range int(cv.Val) {
		iv, err := oleutil.CallMethod(wins, "Item", i)
		if err != nil || iv.VT != ole.VT_DISPATCH || iv.Val == 0 {
			continue
		}
		item := iv.ToIDispatch()
		t := explorerTab{tab: tabWindow(item)}
		if v, err := oleutil.GetProperty(item, "HWND"); err == nil {
			t.hwnd = uintptr(v.Val)
		}
		t.path = folderPath(item)
		item.Release()
		out = append(out, t)
	}
	return out, nil
}

// folderPath = item.Document.Folder.Self.Path
func folderPath(item *ole.IDispatch) string {
	cur := item
	var held []*ole.IDispatch
	defer func() {
		for _, d := range held {
			d.Release()
		}
	}()
	for _, prop := range []string{"Document", "Folder", "Self"} {
		v, err := oleutil.GetProperty(cur, prop)
		if err != nil || v.VT != ole.VT_DISPATCH || v.Val == 0 {
			return ""
		}
		cur = v.ToIDispatch()
		held = append(held, cur)
	}
	v, err := oleutil.GetProperty(cur, "Path")
	if err != nil {
		return ""
	}
	return v.ToString()
}

// ForegroundExplorerFolder returns the folder shown in the foreground
// Explorer window's current tab. ok is false when the foreground window is
// not Explorer.
func ForegroundExplorerFolder() (path string, ok bool, err error) {
	fg, _, _ := procGetForegroundWindow.Call()
	if fg == 0 || className(fg) != "CabinetWClass" {
		return "", false, nil
	}
	tabs, err := explorerTabs()
	if err != nil {
		return "", true, err
	}
	var cands []explorerTab
	for _, t := range tabs {
		if t.hwnd == fg {
			cands = append(cands, t)
		}
	}
	switch len(cands) {
	case 0:
		return "", true, fmt.Errorf("the Explorer window was not found in ShellWindows")
	case 1:
		return cands[0].path, true, nil
	}
	at := activeTab(fg)
	for _, t := range cands {
		if t.tab != 0 && t.tab == at {
			return t.path, true, nil
		}
	}
	return "", true, fmt.Errorf("could not tell which Explorer tab is shown")
}
