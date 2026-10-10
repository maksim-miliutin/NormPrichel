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

func TestTheOverlayGoesWhereToldLetsClicksThroughAndNeverTakesTheFocus(t *testing.T) {
	window := &Window{}
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
	onLoop := func(action func() error) error {
		actions <- action
		return <-results
	}

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
