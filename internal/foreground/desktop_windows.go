package foreground

import (
	"errors"
	"fmt"
	"image"
	"unsafe"

	"golang.org/x/sys/windows"
)

var ErrSnapshot = errors.New("foreground: the running processes cannot be listed")

var (
	user32             = windows.NewLazySystemDLL("user32.dll")
	procIsIconic       = user32.NewProc("IsIconic")
	procGetClientRect  = user32.NewProc("GetClientRect")
	procClientToScreen = user32.NewProc("ClientToScreen")
)

type point struct {
	X, Y int32
}

type System struct{}

func (System) Foreground() Window {
	hwnd := windows.GetForegroundWindow()
	if hwnd == 0 {
		return Window{}
	}

	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(hwnd, &pid); err != nil {
		return Window{}
	}

	client, ok := clientArea(hwnd)
	if !ok {
		return Window{}
	}

	minimized, _, _ := procIsIconic.Call(uintptr(hwnd))

	return Window{Handle: uintptr(hwnd), Process: pid, Minimized: minimized != 0, Client: client}
}

// The rectangle counts physical pixels only once the process has declared per-monitor DPI
// awareness; before that Windows hands out scaled coordinates.
func clientArea(hwnd windows.HWND) (image.Rectangle, bool) {
	var size windows.Rect
	if ok, _, _ := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&size))); ok == 0 {
		return image.Rectangle{}, false
	}

	var origin point
	if ok, _, _ := procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&origin))); ok == 0 {
		return image.Rectangle{}, false
	}

	return image.Rect(int(origin.X), int(origin.Y), int(origin.X+size.Right), int(origin.Y+size.Bottom)), true
}

func (System) Processes() ([]Process, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSnapshot, err)
	}
	defer windows.CloseHandle(snapshot)

	// Windows refuses to walk the list with an entry whose Size is not filled in first.
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}

	var processes []Process
	for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
		processes = append(processes, Process{ID: entry.ProcessID, Exe: windows.UTF16ToString(entry.ExeFile[:])})
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return nil, fmt.Errorf("%w: %w", ErrSnapshot, err)
	}

	return processes, nil
}
