package overlay

import (
	"errors"
	"fmt"
	"image"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ErrDPI    = errors.New("overlay: per-monitor DPI awareness was refused")
	ErrWindow = errors.New("overlay: the overlay window cannot be made")
	ErrLoop   = errors.New("overlay: the message loop broke")
	ErrUpdate = errors.New("overlay: the overlay window refused the picture")
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	gdi32  = windows.NewLazySystemDLL("gdi32.dll")

	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procRegisterClassExW              = user32.NewProc("RegisterClassExW")
	procCreateWindowExW               = user32.NewProc("CreateWindowExW")
	procDefWindowProcW                = user32.NewProc("DefWindowProcW")
	procDestroyWindow                 = user32.NewProc("DestroyWindow")
	procShowWindow                    = user32.NewProc("ShowWindow")
	procSetWindowPos                  = user32.NewProc("SetWindowPos")
	procUpdateLayeredWindow           = user32.NewProc("UpdateLayeredWindow")
	procGetMessageW                   = user32.NewProc("GetMessageW")
	procTranslateMessage              = user32.NewProc("TranslateMessage")
	procDispatchMessageW              = user32.NewProc("DispatchMessageW")
	procPostQuitMessage               = user32.NewProc("PostQuitMessage")
	procPostMessageW                  = user32.NewProc("PostMessageW")
	procSetTimer                      = user32.NewProc("SetTimer")
	procSetWinEventHook               = user32.NewProc("SetWinEventHook")
	procUnhookWinEvent                = user32.NewProc("UnhookWinEvent")
	procCreateCompatibleDC            = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC                      = gdi32.NewProc("DeleteDC")
	procCreateDIBSection              = gdi32.NewProc("CreateDIBSection")
	procSelectObject                  = gdi32.NewProc("SelectObject")
	procDeleteObject                  = gdi32.NewProc("DeleteObject")
)

const (
	wsPopup         = 0x80000000
	wsExLayered     = 0x00080000
	wsExTransparent = 0x00000020
	wsExTopmost     = 0x00000008
	wsExToolWindow  = 0x00000080
	wsExNoActivate  = 0x08000000

	swHide = 0

	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoActivate = 0x0010
	swpShowWindow = 0x0040

	ulwAlpha   = 0x02
	acSrcOver  = 0x00
	acSrcAlpha = 0x01

	wmDestroy = 0x0002
	wmClose   = 0x0010
	wmTimer   = 0x0113

	eventSystemForeground    = 0x0003
	eventSystemMinimizeStart = 0x0016
	eventSystemMinimizeEnd   = 0x0017
	winEventOutOfContext     = 0x0000

	biRGB        = 0
	dibRGBColors = 0

	tickTimer = 1
	tick      = 100 // milliseconds between looks at the game window
)

// Both handles are small negative numbers in the Windows headers: HWND_TOPMOST is -1 and
// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 is -4.
var (
	hwndTopmost       = ^uintptr(0)
	perMonitorAwareV2 = ^uintptr(3)
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

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}

type point struct {
	X, Y int32
}

type extent struct {
	CX, CY int32
}

type blendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

// One overlay per process: the window procedure and the event hook are plain functions handed
// to Windows once, and they reach the running window through these.
var (
	callbacks = sync.OnceValues(func() (uintptr, uintptr) {
		return windows.NewCallback(windowProc), windows.NewCallback(eventProc)
	})
	onEvent func()
)

var (
	className, _ = windows.UTF16PtrFromString("NormPrichelOverlay")
	title, _     = windows.UTF16PtrFromString("NormPrichel")
)

type Window struct {
	hwnd atomic.Uintptr
}

// DeclareDPIAwareness has to come before the first window of the process; without it Windows
// stretches the overlay on a scaled screen, and the crosshair drifts off the centre and blurs.
func DeclareDPIAwareness() error {
	ok, _, err := procSetProcessDpiAwarenessContext.Call(perMonitorAwareV2)
	if ok == 0 && !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		return fmt.Errorf("%w: %w", ErrDPI, err)
	}

	return nil
}

// Run makes the window and pumps its messages until Stop. Windows ties a window, its timer and
// its event hooks to the thread that made them, so refresh always runs on that one thread.
func (w *Window) Run(refresh func()) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hwnd, err := create()
	if err != nil {
		return err
	}
	w.hwnd.Store(hwnd)
	defer w.hwnd.Store(0)

	onEvent = refresh
	defer func() { onEvent = nil }()

	hooks, err := hook()
	defer unhook(hooks)
	if err != nil {
		procDestroyWindow.Call(hwnd)
		return err
	}

	if ok, _, err := procSetTimer.Call(hwnd, tickTimer, tick, 0); ok == 0 {
		procDestroyWindow.Call(hwnd)
		return fmt.Errorf("%w: %w", ErrWindow, err)
	}

	refresh()

	return pump(hwnd, refresh)
}

func (w *Window) Stop() {
	if hwnd := w.hwnd.Load(); hwnd != 0 {
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
	}
}

func (w *Window) Show(bitmap Bitmap, at image.Point) error {
	dc, _, err := procCreateCompatibleDC.Call(0)
	if dc == 0 {
		return fmt.Errorf("%w: %w", ErrUpdate, err)
	}
	defer procDeleteDC.Call(dc)

	info := bitmapInfo{Header: bitmapInfoHeader{
		Width:       int32(bitmap.Size.X),
		Height:      -int32(bitmap.Size.Y),
		Planes:      1,
		BitCount:    32,
		Compression: biRGB,
	}}
	info.Header.Size = uint32(unsafe.Sizeof(info.Header))

	var bits unsafe.Pointer
	section, _, err := procCreateDIBSection.Call(dc, uintptr(unsafe.Pointer(&info)), dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if section == 0 {
		return fmt.Errorf("%w: %w", ErrUpdate, err)
	}
	defer procDeleteObject.Call(section)

	copy(unsafe.Slice((*byte)(bits), len(bitmap.Pixels)), bitmap.Pixels)

	previous, _, _ := procSelectObject.Call(dc, section)
	defer procSelectObject.Call(dc, previous)

	destination := point{X: int32(at.X), Y: int32(at.Y)}
	size := extent{CX: int32(bitmap.Size.X), CY: int32(bitmap.Size.Y)}
	source := point{}
	blend := blendFunction{BlendOp: acSrcOver, SourceConstantAlpha: 255, AlphaFormat: acSrcAlpha}
	ok, _, err := procUpdateLayeredWindow.Call(
		w.hwnd.Load(), 0, uintptr(unsafe.Pointer(&destination)), uintptr(unsafe.Pointer(&size)),
		dc, uintptr(unsafe.Pointer(&source)), 0, uintptr(unsafe.Pointer(&blend)), ulwAlpha,
	)
	if ok == 0 {
		return fmt.Errorf("%w: %w", ErrUpdate, err)
	}

	return w.place(0, 0, swpNoMove|swpNoSize|swpNoActivate|swpShowWindow)
}

func (w *Window) Move(at image.Point) error {
	return w.place(at.X, at.Y, swpNoSize|swpNoActivate)
}

func (w *Window) Hide() error {
	procShowWindow.Call(w.hwnd.Load(), swHide)

	return nil
}

// A borderless game may sit in the topmost band too, and Windows lifts whichever window came
// forward last, so every show and move puts the overlay back at the top of the band.
func (w *Window) place(x, y int, flags uintptr) error {
	ok, _, err := procSetWindowPos.Call(w.hwnd.Load(), hwndTopmost, uintptr(x), uintptr(y), 0, 0, flags)
	if ok == 0 {
		return fmt.Errorf("%w: %w", ErrUpdate, err)
	}

	return nil
}

func create() (uintptr, error) {
	var instance windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
		return 0, fmt.Errorf("%w: %w", ErrWindow, err)
	}

	procedure, _ := callbacks()
	wc := wndClassEx{WndProc: procedure, Instance: instance, ClassName: className}
	wc.Size = uint32(unsafe.Sizeof(wc))
	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 && !errors.Is(err, windows.ERROR_CLASS_ALREADY_EXISTS) {
		return 0, fmt.Errorf("%w: %w", ErrWindow, err)
	}

	hwnd, _, err := procCreateWindowExW.Call(
		wsExLayered|wsExTransparent|wsExTopmost|wsExToolWindow|wsExNoActivate,
		uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)),
		wsPopup, 0, 0, 0, 0, 0, 0, uintptr(instance), 0,
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("%w: %w", ErrWindow, err)
	}

	return hwnd, nil
}

func hook() ([]uintptr, error) {
	_, events := callbacks()

	var hooks []uintptr
	for _, span := range [][2]uintptr{
		{eventSystemForeground, eventSystemForeground},
		{eventSystemMinimizeStart, eventSystemMinimizeEnd},
	} {
		h, _, err := procSetWinEventHook.Call(span[0], span[1], 0, events, 0, 0, winEventOutOfContext)
		if h == 0 {
			return hooks, fmt.Errorf("%w: %w", ErrWindow, err)
		}
		hooks = append(hooks, h)
	}

	return hooks, nil
}

func unhook(hooks []uintptr) {
	for _, h := range hooks {
		procUnhookWinEvent.Call(h)
	}
}

func pump(hwnd uintptr, refresh func()) error {
	var m msg
	for {
		got, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		switch int32(got) {
		case -1:
			return fmt.Errorf("%w: %w", ErrLoop, err)
		case 0:
			return nil
		}

		if m.Message == wmTimer && m.Hwnd == hwnd {
			refresh()
			continue
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func windowProc(hwnd, message, wparam, lparam uintptr) uintptr {
	if message == wmDestroy {
		procPostQuitMessage.Call(0)
		return 0
	}

	result, _, _ := procDefWindowProcW.Call(hwnd, message, wparam, lparam)

	return result
}

func eventProc(_, _, _, _, _, _, _ uintptr) uintptr {
	if onEvent != nil {
		onEvent()
	}

	return 0
}
