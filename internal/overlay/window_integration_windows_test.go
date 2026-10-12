//go:build integration

package overlay

import (
	"image"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procGetFocus                     = user32.NewProc("GetFocus")
	procSetFocus                     = user32.NewProc("SetFocus")
	procGetWindowLongPtrW            = user32.NewProc("GetWindowLongPtrW")
	procGetWindowRect                = user32.NewProc("GetWindowRect")
	procWindowFromPoint              = user32.NewProc("WindowFromPoint")
	procGetThreadDpiAwarenessContext = user32.NewProc("GetThreadDpiAwarenessContext")
	procAreDpiAwarenessContextsEqual = user32.NewProc("AreDpiAwarenessContextsEqual")
)

// GWL_EXSTYLE is -20 in the Windows headers.
var gwlExStyle = ^uintptr(19)

func TestTheProcessBecomesPerMonitorAware(t *testing.T) {
	if err := DeclareDPIAwareness(); err != nil {
		t.Fatal(err)
	}

	current, _, _ := procGetThreadDpiAwarenessContext.Call()
	if same, _, _ := procAreDpiAwarenessContextsEqual.Call(current, perMonitorAwareV2); same == 0 {
		t.Error("the thread is not per-monitor aware v2 after the declaration")
	}
}

// running starts the window's loop and hands back a way to run code on the loop's own thread,
// where the windows it serves live.
func running(window *Window) (func(action func() error) error, chan error) {
	actions := make(chan func() error)
	results := make(chan error)
	stopped := make(chan error, 1)
	go func() {
		stopped <- window.Run(func() {
			select {
			case action := <-actions:
				results <- action()
			default:
			}
		})
	}()

	return func(action func() error) error {
		actions <- action
		return <-results
	}, stopped
}

func TestTheOverlayGoesWhereToldLetsClicksThroughAndNeverTakesTheFocus(t *testing.T) {
	window := &Window{}
	onLoop, stopped := running(window)

	red := Bitmap{Pixels: make([]byte, 4*4*4), Size: image.Pt(4, 4)}
	for i := 0; i < len(red.Pixels); i += 4 {
		red.Pixels[i+2], red.Pixels[i+3] = 255, 255
	}
	if err := onLoop(func() error { return window.Show(red, image.Pt(100, 120)) }); err != nil {
		t.Fatal(err)
	}
	hwnd := window.hwnd.Load()

	extended, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle)
	for name, flag := range map[string]uintptr{
		"layered":                 wsExLayered,
		"click-through":           wsExTransparent,
		"topmost":                 wsExTopmost,
		"out of Alt+Tab":          wsExToolWindow,
		"never the active window": wsExNoActivate,
	} {
		if extended&flag == 0 {
			t.Errorf("the overlay is not %s", name)
		}
	}

	if !windows.IsWindowVisible(windows.HWND(hwnd)) {
		t.Error("the overlay is hidden after Show")
	}
	if got, want := rectOf(hwnd), (windows.Rect{Left: 100, Top: 120, Right: 104, Bottom: 124}); got != want {
		t.Errorf("the overlay covers %+v, want %+v", got, want)
	}
	if under, _, _ := procWindowFromPoint.Call(uintptr(uint32(101)) | uintptr(uint32(121))<<32); under == hwnd {
		t.Error("a click on the crosshair lands on the overlay instead of the window beneath")
	}
	if windows.GetForegroundWindow() == windows.HWND(hwnd) {
		t.Error("the overlay took the focus")
	}

	if err := onLoop(func() error { return window.Move(image.Pt(-50, 30)) }); err != nil {
		t.Fatal(err)
	}
	if got, want := rectOf(hwnd), (windows.Rect{Left: -50, Top: 30, Right: -46, Bottom: 34}); got != want {
		t.Errorf("after the move the overlay covers %+v, want %+v", got, want)
	}

	if err := onLoop(window.Hide); err != nil {
		t.Fatal(err)
	}
	if windows.IsWindowVisible(windows.HWND(hwnd)) {
		t.Error("the overlay is visible after Hide")
	}

	window.Stop()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return within 5 s of Stop")
	}
}

func rectOf(hwnd uintptr) windows.Rect {
	var rect windows.Rect
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))

	return rect
}

func TestTheLoopGivesTheDialogWindowTabBetweenItsControls(t *testing.T) {
	var dialog, first, second uintptr
	window := &Window{Dialog: func() uintptr { return dialog }}
	onLoop, stopped := running(window)
	defer func() {
		window.Stop()
		<-stopped
	}()

	create := func(class string, style, x, parent uintptr) uintptr {
		name, _ := windows.UTF16PtrFromString(class)
		hwnd, _, _ := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(name)), 0, style|0x10000000, x, 0, 50, 20, parent, 0, 0, 0)
		return hwnd
	}
	onLoop(func() error {
		dialog = create("STATIC", 0x00CF0000, 0, 0)
		first = create("BUTTON", 0x40010000, 0, dialog)
		second = create("BUTTON", 0x40010000, 60, dialog)
		procSetFocus.Call(first)
		procPostMessageW.Call(first, 0x0100, 0x09, 0)
		return nil
	})

	var focus uintptr
	onLoop(func() error {
		focus, _, _ = procGetFocus.Call()
		procDestroyWindow.Call(dialog)
		return nil
	})
	if focus != second || second == 0 {
		t.Errorf("after Tab the keys are with %#x, want the second button %#x", focus, second)
	}
}
