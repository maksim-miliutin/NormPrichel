//go:build integration

package tray_test

import (
	"runtime"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/maksim-miliutin/NormPrichel/internal/tray"
)

var (
	user32              = windows.NewLazySystemDLL("user32.dll")
	procPostMessageW    = user32.NewProc("PostMessageW")
	procPeekMessageW    = user32.NewProc("PeekMessageW")
	procDispatchMessage = user32.NewProc("DispatchMessageW")
	procGetWindowLongW  = user32.NewProc("GetWindowLongPtrW")
	procFindWindowW     = user32.NewProc("FindWindowW")
)

const (
	wmHotkey       = 0x0312
	pmRemove       = 0x0001
	wsExToolWindow = 0x00000080
)

// GWL_EXSTYLE is -20 in the Windows headers.
var gwlExStyle = ^uintptr(19)

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
	Private uint32
}

// The tests make the program's own window, and a running copy would answer in its place.
func alone(t *testing.T) {
	t.Helper()

	name, _ := windows.UTF16PtrFromString("NormPrichel")
	if found, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(name)), 0); found != 0 {
		t.Fatal("NormPrichel is running: close it from its tray icon before these tests")
	}
}

func TestAHotkeyPressReachesItsHandlerAndTheWindowStaysOutOfSight(t *testing.T) {
	alone(t)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	pressed := -1
	window, err := tray.New("en", tray.Handlers{Hotkey: func(action int) { pressed = action }})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()

	procPostMessageW.Call(window.Handle(), wmHotkey, 7, 0)
	deadline := time.Now().Add(2 * time.Second)
	for pressed < 0 && time.Now().Before(deadline) {
		var m msg
		if got, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove); got != 0 {
			procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
		}
	}
	if pressed != 7 {
		t.Errorf("the handler saw action %d, want 7", pressed)
	}

	if windows.IsWindowVisible(windows.HWND(window.Handle())) {
		t.Error("the window is visible")
	}
	if extended, _, _ := procGetWindowLongW.Call(window.Handle(), gwlExStyle); extended&wsExToolWindow == 0 {
		t.Error("the window would show in Alt+Tab")
	}

	name, _ := windows.UTF16PtrFromString("NormPrichel")
	if found, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(name)), 0); found != window.Handle() {
		t.Errorf("a second copy of the program would find window %#x, want %#x", found, window.Handle())
	}
}

func TestTheIconGoesIntoTheNotificationAreaAndLeavesWithTheWindow(t *testing.T) {
	alone(t)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	window, err := tray.New("ru", tray.Handlers{})
	if err != nil {
		t.Fatal(err)
	}
	if err := window.ShowIcon(); err != nil {
		window.Close()
		t.Fatal(err)
	}
	t.Log("the crosshair icon is in the notification area for two seconds")
	time.Sleep(2 * time.Second)
	window.Close()
}

func TestTheLanguageOfWindowsIsReadAsRussianOrEnglish(t *testing.T) {
	if got := tray.SystemLanguage(); got != "ru" && got != "en" {
		t.Errorf("got %q, want ru or en", got)
	}
	t.Logf("this Windows speaks %s", tray.SystemLanguage())
}

func TestASecondCopyFindsTheFirstAndAsksItToShowItsSettings(t *testing.T) {
	alone(t)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	first, err := tray.Claim()
	if err != nil {
		t.Fatal(err)
	}
	if !first {
		t.Fatal("the program is running: close it before this test")
	}
	if again, err := tray.Claim(); err != nil || again {
		t.Errorf("a second claim: %v, %v; want it refused", again, err)
	}

	opened := false
	window, err := tray.New("en", tray.Handlers{Open: func() { opened = true }})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()

	woken := make(chan error, 1)
	go func() { woken <- tray.Wake() }()
	deadline := time.Now().Add(5 * time.Second)
	for !opened && time.Now().Before(deadline) {
		var m msg
		if got, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove); got != 0 {
			procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
		}
	}
	if err := <-woken; err != nil || !opened {
		t.Errorf("wake: %v, settings shown %v; want the first copy asked to show them", err, opened)
	}
}

func TestAnIconComesAtTheSizeAskedFor(t *testing.T) {
	getIconInfo := user32.NewProc("GetIconInfo")
	destroyIcon := user32.NewProc("DestroyIcon")
	gdi32 := windows.NewLazySystemDLL("gdi32.dll")
	getObject, deleteObject := gdi32.NewProc("GetObjectW"), gdi32.NewProc("DeleteObject")

	for _, side := range []int{16, 24, 32, 48} {
		icon, err := tray.NewIcon(side)
		if err != nil {
			t.Fatal(err)
		}
		var info struct {
			IsIcon      int32
			XHotspot    uint32
			YHotspot    uint32
			Mask, Color uintptr
		}
		getIconInfo.Call(icon, uintptr(unsafe.Pointer(&info)))
		for name, part := range map[string]uintptr{"colour": info.Color, "mask": info.Mask} {
			var bitmap struct {
				Type, Width, Height, WidthBytes int32
				Planes, BitsPixel               uint16
				Bits                            uintptr
			}
			getObject.Call(part, unsafe.Sizeof(bitmap), uintptr(unsafe.Pointer(&bitmap)))
			deleteObject.Call(part)
			if bitmap.Width != int32(side) || bitmap.Height != int32(side) {
				t.Errorf("asked for %d, got a %s of %d by %d", side, name, bitmap.Width, bitmap.Height)
			}
		}
		destroyIcon.Call(icon)

		if info.IsIcon == 0 {
			t.Errorf("asked for %d, got a cursor", side)
		}
	}
}
