package tray

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var ErrWindow = errors.New("tray: the program's window cannot be made")

var (
	user32 = windows.NewLazySystemDLL("user32.dll")

	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
)

const (
	wsPopup        = 0x80000000
	wsExToolWindow = 0x00000080

	wmHotkey = 0x0312
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   windows.Handle
	Icon       windows.Handle
	Cursor     windows.Handle
	Background windows.Handle
	MenuName   *uint16
	ClassName  *uint16
	IconSm     windows.Handle
}

type Handlers struct {
	Hotkey func(action int)
}

// One window per process: the window procedure is a plain function handed to Windows once, and
// it reaches the handlers through these.
var (
	procedure = sync.OnceValue(func() uintptr { return windows.NewCallback(windowProc) })
	handlers  Handlers

	className, _ = windows.UTF16PtrFromString("NormPrichel")
)

// Window is never shown: it is what hotkeys and the tray icon talk to. Its messages arrive
// through the message loop of the thread that made it.
type Window struct {
	hwnd uintptr
}

func New(h Handlers) (*Window, error) {
	var instance windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWindow, err)
	}

	wc := wndClassEx{WndProc: procedure(), Instance: instance, ClassName: className}
	wc.Size = uint32(unsafe.Sizeof(wc))
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 && !errors.Is(err, windows.ERROR_CLASS_ALREADY_EXISTS) {
		return nil, fmt.Errorf("%w: %w", ErrWindow, err)
	}

	handlers = h
	hwnd, _, err := procCreateWindowExW.Call(
		wsExToolWindow, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(className)),
		wsPopup, 0, 0, 0, 0, 0, 0, uintptr(instance), 0,
	)
	if hwnd == 0 {
		return nil, fmt.Errorf("%w: %w", ErrWindow, err)
	}

	return &Window{hwnd: hwnd}, nil
}

func (w *Window) Handle() uintptr {
	return w.hwnd
}

func (w *Window) Close() {
	procDestroyWindow.Call(w.hwnd)
	handlers = Handlers{}
}

func windowProc(hwnd, message, wparam, lparam uintptr) uintptr {
	if message == wmHotkey && handlers.Hotkey != nil {
		handlers.Hotkey(int(wparam))
		return 0
	}

	result, _, _ := procDefWindowProcW.Call(hwnd, message, wparam, lparam)

	return result
}
