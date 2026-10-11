package tray

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ErrWindow = errors.New("tray: the program's window cannot be made")
	ErrIcon   = errors.New("tray: the icon cannot be put in the notification area")
)

var (
	user32  = windows.NewLazySystemDLL("user32.dll")
	gdi32   = windows.NewLazySystemDLL("gdi32.dll")
	shell32 = windows.NewLazySystemDLL("shell32.dll")

	procRegisterClassExW         = user32.NewProc("RegisterClassExW")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDefWindowProcW           = user32.NewProc("DefWindowProcW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procRegisterWindowMessageW   = user32.NewProc("RegisterWindowMessageW")
	procCreatePopupMenu          = user32.NewProc("CreatePopupMenu")
	procAppendMenuW              = user32.NewProc("AppendMenuW")
	procTrackPopupMenu           = user32.NewProc("TrackPopupMenu")
	procDestroyMenu              = user32.NewProc("DestroyMenu")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procPostMessageW             = user32.NewProc("PostMessageW")
	procCreateIconIndirect       = user32.NewProc("CreateIconIndirect")
	procDestroyIcon              = user32.NewProc("DestroyIcon")
	procGetSystemMetricsForDpi   = user32.NewProc("GetSystemMetricsForDpi")
	procGetDpiForSystem          = user32.NewProc("GetDpiForSystem")
	procCreateDIBSection         = gdi32.NewProc("CreateDIBSection")
	procCreateBitmap             = gdi32.NewProc("CreateBitmap")
	procDeleteObject             = gdi32.NewProc("DeleteObject")
	procShellNotifyIconW         = shell32.NewProc("Shell_NotifyIconW")
	procGetUserDefaultUILanguage = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")
)

const (
	wsPopup        = 0x80000000
	wsExToolWindow = 0x00000080

	wmNull          = 0x0000
	wmContextMenu   = 0x007B
	wmHotkey        = 0x0312
	wmLButtonDblClk = 0x0203
	wmRButtonUp     = 0x0205
	wmTray          = 0x8000 + 1

	nimAdd     = 0
	nimDelete  = 2
	nifMessage = 0x1
	nifIcon    = 0x2
	nifTip     = 0x4

	mfString    = 0x0000
	mfChecked   = 0x0008
	mfSeparator = 0x0800

	tpmRightButton = 0x0002
	tpmNoNotify    = 0x0080
	tpmReturnCmd   = 0x0100

	smCxSmIcon = 49

	biRGB        = 0
	dibRGBColors = 0
)

const (
	commandSettings = 1 + iota
	commandToggle
	commandAutostart
	commandExit
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

type notifyIconData struct {
	Size            uint32
	Window          uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Version         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GUIDItem        windows.GUID
	BalloonIcon     uintptr
}

type iconInfo struct {
	IsIcon   int32
	XHotspot uint32
	YHotspot uint32
	Mask     uintptr
	Color    uintptr
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

type point struct {
	X, Y int32
}

type Handlers struct {
	Hotkey func(action int)
	Open   func()
	Toggle func()
	Exit   func()
	Fail   func(err error)
}

// One window per process: the window procedure is a plain function handed to Windows once, and
// it reaches the running window through these.
var (
	procedure = sync.OnceValue(func() uintptr { return windows.NewCallback(windowProc) })
	running   *Window

	className, _ = windows.UTF16PtrFromString("NormPrichel")
)

// Window is never shown: hotkeys and the tray icon talk to it, and its messages arrive through
// the message loop of the thread that made it.
type Window struct {
	hwnd     uintptr
	icon     uintptr
	labels   Labels
	handlers Handlers
	restart  uint32
}

func New(language string, h Handlers) (*Window, error) {
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

	restart, _ := windows.UTF16PtrFromString("TaskbarCreated")
	message, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(restart)))
	w := &Window{labels: LabelsFor(language), handlers: h, restart: uint32(message)}
	running = w

	hwnd, _, err := procCreateWindowExW.Call(
		wsExToolWindow, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(className)),
		wsPopup, 0, 0, 0, 0, 0, 0, uintptr(instance), 0,
	)
	if hwnd == 0 {
		running = nil
		return nil, fmt.Errorf("%w: %w", ErrWindow, err)
	}
	w.hwnd = hwnd

	return w, nil
}

func (w *Window) Handle() uintptr {
	return w.hwnd
}

// ShowIcon puts the crosshair in the notification area; Explorer forgets it when it restarts,
// and the window puts it back when Explorer says it is up again.
func (w *Window) ShowIcon() error {
	if w.icon == 0 {
		icon, err := makeIcon()
		if err != nil {
			return err
		}
		w.icon = icon
	}

	data := w.notifyData()
	data.Flags = nifMessage | nifIcon | nifTip
	data.CallbackMessage = wmTray
	data.Icon = w.icon
	tip, _ := windows.UTF16FromString("NormPrichel")
	copy(data.Tip[:len(data.Tip)-1], tip)
	if ok, _, err := procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&data))); ok == 0 {
		return fmt.Errorf("%w: %w", ErrIcon, err)
	}

	return nil
}

func (w *Window) Close() {
	data := w.notifyData()
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&data)))
	if w.icon != 0 {
		procDestroyIcon.Call(w.icon)
	}
	procDestroyWindow.Call(w.hwnd)
	running = nil
}

func (w *Window) notifyData() notifyIconData {
	data := notifyIconData{Window: w.hwnd, ID: 1}
	data.Size = uint32(unsafe.Sizeof(data))

	return data
}

// The menu goes away on a click elsewhere only when its window is in front, so the hidden window
// comes forward first, and the empty message after it is what Windows asks for in return.
func (w *Window) showMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	starts, err := Autostart()
	w.fail(err)
	ticked := uintptr(mfString)
	if starts {
		ticked |= mfChecked
	}

	for _, item := range []struct {
		id    uintptr
		flags uintptr
		text  string
	}{
		{commandSettings, mfString, w.labels.Settings},
		{commandToggle, mfString, w.labels.Toggle},
		{commandAutostart, ticked, w.labels.Autostart},
		{0, mfSeparator, ""},
		{commandExit, mfString, w.labels.Exit},
	} {
		text, _ := windows.UTF16PtrFromString(item.text)
		procAppendMenuW.Call(menu, item.flags, item.id, uintptr(unsafe.Pointer(text)))
	}

	var at point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&at)))
	procSetForegroundWindow.Call(w.hwnd)
	command, _, _ := procTrackPopupMenu.Call(menu, tpmRightButton|tpmNoNotify|tpmReturnCmd, uintptr(at.X), uintptr(at.Y), 0, w.hwnd, 0)
	procPostMessageW.Call(w.hwnd, wmNull, 0, 0)

	if command == commandAutostart {
		w.fail(SetAutostart(!starts))
		return
	}
	w.run(command)
}

func (w *Window) run(command uintptr) {
	actions := map[uintptr]func(){commandSettings: w.handlers.Open, commandToggle: w.handlers.Toggle, commandExit: w.handlers.Exit}
	if action := actions[command]; action != nil {
		action()
	}
}

func (w *Window) fail(err error) {
	if err != nil && w.handlers.Fail != nil {
		w.handlers.Fail(err)
	}
}

func windowProc(hwnd, message, wparam, lparam uintptr) uintptr {
	w := running
	switch {
	case w == nil:
	case message == wmHotkey && w.handlers.Hotkey != nil:
		w.handlers.Hotkey(int(wparam))
		return 0
	case message == wmTray && lparam&0xFFFF == wmLButtonDblClk, message == wmWake:
		w.run(commandSettings)
		return 0
	case message == wmTray && (lparam&0xFFFF == wmRButtonUp || lparam&0xFFFF == wmContextMenu):
		w.showMenu()
		return 0
	case message == uintptr(w.restart) && w.restart != 0:
		w.ShowIcon()
		return 0
	}

	result, _, _ := procDefWindowProcW.Call(hwnd, message, wparam, lparam)

	return result
}

func makeIcon() (uintptr, error) {
	dpi, _, _ := procGetDpiForSystem.Call()
	side, _, _ := procGetSystemMetricsForDpi.Call(smCxSmIcon, dpi)
	if side == 0 {
		side = 16
	}

	pixels, err := Icon(int(side))
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrIcon, err)
	}

	header := bitmapInfoHeader{Width: int32(side), Height: -int32(side), Planes: 1, BitCount: 32, Compression: biRGB}
	header.Size = uint32(unsafe.Sizeof(header))
	var bits unsafe.Pointer
	color, _, err := procCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&header)), dibRGBColors, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if color == 0 {
		return 0, fmt.Errorf("%w: %w", ErrIcon, err)
	}
	defer procDeleteObject.Call(color)
	copy(unsafe.Slice((*byte)(bits), len(pixels)), pixels)

	// The colour bitmap carries alpha, so the mask only has to exist: all zero, rows padded to words.
	mask := make([]byte, int(side)*((int(side)+15)/16*2))
	monochrome, _, err := procCreateBitmap.Call(side, side, 1, 1, uintptr(unsafe.Pointer(&mask[0])))
	if monochrome == 0 {
		return 0, fmt.Errorf("%w: %w", ErrIcon, err)
	}
	defer procDeleteObject.Call(monochrome)

	info := iconInfo{IsIcon: 1, Mask: monochrome, Color: color}
	icon, _, err := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&info)))
	if icon == 0 {
		return 0, fmt.Errorf("%w: %w", ErrIcon, err)
	}

	return icon, nil
}

func SystemLanguage() string {
	id, _, _ := procGetUserDefaultUILanguage.Call()

	return Language(uint16(id))
}

func Open(address string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	target, err := windows.UTF16PtrFromString(address)
	if err != nil {
		return err
	}

	return windows.ShellExecute(0, verb, target, nil, nil, windows.SW_SHOWNORMAL)
}
