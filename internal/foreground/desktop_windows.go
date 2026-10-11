package foreground

import (
	"errors"
	"fmt"
	"image"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ErrSnapshot = errors.New("foreground: the running processes cannot be listed")
	ErrWindows  = errors.New("foreground: the windows on the desktop cannot be listed")
)

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	procIsIconic          = user32.NewProc("IsIconic")
	procGetClientRect     = user32.NewProc("GetClientRect")
	procClientToScreen    = user32.NewProc("ClientToScreen")
	procGetWindowTextW    = user32.NewProc("GetWindowTextW")
	procGetWindowLongPtrW = user32.NewProc("GetWindowLongPtrW")
	procGetWindow         = user32.NewProc("GetWindow")
)

const gwOwner = 4

// GWL_EXSTYLE is -20 in the Windows headers.
var gwlExStyle = ^uintptr(19)

// EnumWindows calls back on the thread that asked, and one callback serves every call, so what
// it collects goes through a variable under a lock.
var (
	enumerating sync.Mutex
	enumerated  []windows.HWND
	collect     = sync.OnceValue(func() uintptr { return windows.NewCallback(collected) })
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

func (s System) Choices(own uint32) ([]Choice, error) {
	top, err := topWindows()
	if err != nil {
		return nil, err
	}

	processes, err := s.Processes()
	if err != nil {
		return nil, err
	}

	return Choices(top, processes, own), nil
}

func topWindows() ([]TopWindow, error) {
	enumerating.Lock()
	enumerated = nil
	err := windows.EnumWindows(collect(), nil)
	found := enumerated
	enumerated = nil
	enumerating.Unlock()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWindows, err)
	}

	desktop := windows.GetShellWindow()
	top := make([]TopWindow, len(found))
	for i, hwnd := range found {
		top[i] = described(hwnd)
		top[i].Desktop = hwnd == desktop
	}

	return top, nil
}

func collected(hwnd windows.HWND, _ uintptr) uintptr {
	enumerated = append(enumerated, hwnd)

	return 1
}

// The title of another program's window is read from Windows itself, not asked of the window,
// so a game that hangs cannot hang the list.
func described(hwnd windows.HWND) TopWindow {
	var pid uint32
	windows.GetWindowThreadProcessId(hwnd, &pid)

	var title [256]uint16
	n, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&title[0])), uintptr(len(title)))
	extended, _, _ := procGetWindowLongPtrW.Call(uintptr(hwnd), gwlExStyle)
	owner, _, _ := procGetWindow.Call(uintptr(hwnd), gwOwner)

	var cloaked uint32
	windows.DwmGetWindowAttribute(hwnd, windows.DWMWA_CLOAKED, unsafe.Pointer(&cloaked), uint32(unsafe.Sizeof(cloaked)))

	return TopWindow{
		Title:    windows.UTF16ToString(title[:n]),
		Process:  pid,
		Visible:  windows.IsWindowVisible(hwnd),
		Extended: uint32(extended),
		Cloaked:  cloaked != 0,
		Owned:    owner != 0 && windows.IsWindowVisible(windows.HWND(owner)),
	}
}
