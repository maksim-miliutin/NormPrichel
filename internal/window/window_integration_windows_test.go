//go:build integration

package window_test

import (
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
	"github.com/maksim-miliutin/NormPrichel/internal/window"
)

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	procGetDlgItem        = user32.NewProc("GetDlgItem")
	procSendMessageW      = user32.NewProc("SendMessageW")
	procIsWindow          = user32.NewProc("IsWindow")
	procIsWindowVisible   = user32.NewProc("IsWindowVisible")
	procGetClassNameW     = user32.NewProc("GetClassNameW")
	procGetWindowTextW    = user32.NewProc("GetWindowTextW")
	procSetWindowTextW    = user32.NewProc("SetWindowTextW")
	procGetKeyboardState  = user32.NewProc("GetKeyboardState")
	procSetKeyboardState  = user32.NewProc("SetKeyboardState")
	procGetWindowLongPtrW = user32.NewProc("GetWindowLongPtrW")
	procSetFocus          = user32.NewProc("SetFocus")
	procLoadIconW         = user32.NewProc("LoadIconW")
)

const (
	idOK            = 1
	idCancel        = 2
	idiApplication  = 32512
	wmGetIcon       = 0x007F
	bnDoubleClicked = 5
	wmClose         = 0x0010
	wmKeyDown       = 0x0100
	wmCommand       = 0x0111
	bmClick         = 0x00F5
	lbSetCurSel     = 0x0186
	enKillFocus     = 0x0200
	lbnSelChange    = 1
	vkRight         = 0x27
	vkControl       = 0x11
	vkMenu          = 0x12
)

// GWL_EXSTYLE is -20 in the Windows headers.
var gwlExStyle = ^uintptr(19)

type opened struct {
	window  *window.Window
	actions window.Actions
	state   *changes
}

// open makes the window on this test's own thread: a window answers only on the thread that made it.
func open(t *testing.T, s profiles.Settings) opened {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)

	a, state, _, _ := newActions(t, s)
	w, err := window.New(a.Form, func() {}, func(err error) { t.Log(err) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.Close)

	return opened{window: w, actions: a, state: state}
}

func (o opened) child(t *testing.T, match func(window.Control) bool) (uintptr, int) {
	t.Helper()
	id := control(t, o.actions, match)
	hwnd, _, _ := procGetDlgItem.Call(o.window.Handle(), uintptr(id))
	if hwnd == 0 {
		t.Fatalf("control %d is not in the window", id)
	}

	return hwnd, id
}

func (o opened) command(id, code int, child uintptr) {
	procSendMessageW.Call(o.window.Handle(), wmCommand, uintptr(code<<16|id), child)
}

func textOf(hwnd uintptr) string {
	buffer := make([]uint16, 256)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))

	return windows.UTF16ToString(buffer)
}

func classOf(hwnd uintptr) string {
	buffer := make([]uint16, 64)
	procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))

	return windows.UTF16ToString(buffer)
}

func TestTheWindowWaitsHiddenShowsWhenAskedAndHidesOnClose(t *testing.T) {
	o := open(t, listed())
	hwnd := o.window.Handle()

	if visible, _, _ := procIsWindowVisible.Call(hwnd); visible != 0 {
		t.Fatal("the window shows before it is asked to")
	}
	o.window.Show()
	if visible, _, _ := procIsWindowVisible.Call(hwnd); visible == 0 {
		t.Fatal("the window stays hidden when asked to show")
	}

	procSendMessageW.Call(hwnd, wmClose, 0, 0)
	visible, _, _ := procIsWindowVisible.Call(hwnd)
	alive, _, _ := procIsWindow.Call(hwnd)
	if visible != 0 || alive == 0 {
		t.Errorf("after closing: visible %v, alive %v; want hidden and alive", visible != 0, alive != 0)
	}
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle)
	if style&0x80 != 0 {
		t.Error("the settings window is a tool window, which keeps it out of the taskbar and Alt+Tab")
	}
}

func TestEveryControlIsMadeOfItsKind(t *testing.T) {
	o := open(t, listed())
	classes := map[window.Kind]string{
		window.Button: "Button", window.Choice: "Button", window.Switch: "Button", window.Colour: "Button", window.Swatch: "Button",
		window.Edit: "Edit", window.Hotkey: "Edit", window.List: "ListBox", window.Label: "Static",
		window.Heading: "Static", window.Hint: "Static", window.Problem: "Static", window.Preview: "Static",
		window.Slider: "NormPrichelSlider",
	}

	for _, p := range window.Places() {
		hwnd, _, _ := procGetDlgItem.Call(o.window.Handle(), uintptr(p.ID))
		if got := classOf(hwnd); hwnd == 0 || !strings.EqualFold(got, classes[p.Kind]) {
			t.Errorf("control %d of kind %s is a %q", p.ID, p.Kind, got)
		}
	}
}

func TestAClickOnAShapeButtonDrawsThatShape(t *testing.T) {
	o := open(t, listed())
	button, _ := o.child(t, reading(o.actions.Form.Word("shape.circle")))

	procSendMessageW.Call(button, bmClick, 0, 0)

	if got := first(t, o.state.Current()).Crosshair.Shape; got != crosshair.Circle {
		t.Errorf("shape %s after the click, want circle", got)
	}
}

func TestAnOffsetTypedIntoItsFieldAppliesWhenTheFieldIsLeft(t *testing.T) {
	o := open(t, listed())
	field, id := o.child(t, func(c window.Control) bool { return c.Kind == window.Edit && c.Text == "0" })

	words, _ := windows.UTF16PtrFromString("12")
	procSetWindowTextW.Call(field, uintptr(unsafe.Pointer(words)))
	o.command(id, enKillFocus, field)

	if got := first(t, o.state.Current()).Offset.X; got != 12 {
		t.Errorf("x %d after leaving the field, want 12", got)
	}
}

func TestAKeyOnASliderMovesItsKnobByAStep(t *testing.T) {
	o := open(t, listed())
	slider, _ := o.child(t, func(c window.Control) bool { return c.Kind == window.Slider && c.Knob.Name == "gap" })
	before := first(t, o.state.Current()).Crosshair.Gap

	procSendMessageW.Call(slider, wmKeyDown, vkRight, 0)

	if got := first(t, o.state.Current()).Crosshair.Gap; got != before+1 {
		t.Errorf("gap %d after the right arrow, want %d", got, before+1)
	}
}

func TestACombinationPressedInAHotkeyFieldBecomesTheHotkeyAndShows(t *testing.T) {
	o := open(t, listed())
	o.window.Show()
	field, _ := o.child(t, nth(window.Hotkey, 0))
	procSetFocus.Call(field)

	var keys [256]byte
	procGetKeyboardState.Call(uintptr(unsafe.Pointer(&keys)))
	t.Cleanup(func() { procSetKeyboardState.Call(uintptr(unsafe.Pointer(&keys))) })
	held := keys
	held[vkControl], held[vkMenu] = 0x80, 0x80
	procSetKeyboardState.Call(uintptr(unsafe.Pointer(&held)))

	procSendMessageW.Call(field, wmKeyDown, 'K', 0)

	if got := o.state.Current().Hotkeys.Toggle; got != "Ctrl+Alt+K" {
		t.Errorf("toggle %q, want Ctrl+Alt+K", got)
	}
	if got := textOf(field); got != "Ctrl+Alt+K" {
		t.Errorf("the field reads %q, want Ctrl+Alt+K", got)
	}
}

func TestChoosingAGameRowShowsItsName(t *testing.T) {
	o := open(t, listed())
	list, id := o.child(t, nth(window.List, 0))
	name, _ := o.child(t, reading("STALZONE"))

	procSendMessageW.Call(list, lbSetCurSel, 1, 0)
	o.command(id, lbnSelChange, list)

	if got := textOf(name); got != "Other" {
		t.Errorf("the name field reads %q, want Other", got)
	}
}

func TestASecondQuickClickOnThePadMovesTheCrosshairAgain(t *testing.T) {
	o := open(t, listed())
	right, id := o.child(t, reading("→"))

	procSendMessageW.Call(right, bmClick, 0, 0)
	o.command(id, bnDoubleClicked, right)

	if got := first(t, o.state.Current()).Offset.X; got != 2 {
		t.Errorf("x %d after a click and a double click, want 2", got)
	}
}

func TestEnterFinishesTheFieldAndEscPutsTheWindowAway(t *testing.T) {
	o := open(t, listed())
	o.window.Show()
	field, _ := o.child(t, func(c window.Control) bool { return c.Kind == window.Edit && c.Text == "0" })
	procSetFocus.Call(field)
	words, _ := windows.UTF16PtrFromString("7.6")
	procSetWindowTextW.Call(field, uintptr(unsafe.Pointer(words)))

	o.command(idOK, 0, 0)
	if got := first(t, o.state.Current()).Offset.X; got != 8 || textOf(field) != "8" {
		t.Fatalf("x %d reading %q after Enter, want 8 in both", got, textOf(field))
	}

	o.command(idCancel, 0, 0)
	if visible, _, _ := procIsWindowVisible.Call(o.window.Handle()); visible != 0 {
		t.Error("the window stays after Esc")
	}
}

func TestTheWindowTakesTheIconAtTheSizesOfItsTitleAndOfAltTab(t *testing.T) {
	o := open(t, listed())
	shared, _, _ := procLoadIconW.Call(0, idiApplication)
	var sides []int

	err := o.window.SetIcon(func(side int) (uintptr, error) {
		sides = append(sides, side)
		return shared, nil
	})

	if err != nil || len(sides) != 2 || sides[0] >= sides[1] {
		t.Fatalf("asked for sides %v, %v; want a small one and a big one", sides, err)
	}
	for which := range uintptr(2) {
		if icon, _, _ := procSendMessageW.Call(o.window.Handle(), wmGetIcon, which, 0); icon != shared {
			t.Errorf("icon %d is %#x, want %#x", which, icon, shared)
		}
	}
}

func TestAFieldBeingTypedInKeepsItsWordsWhileSomethingElseChanges(t *testing.T) {
	o := open(t, listed())
	o.window.Show()
	name, _ := o.child(t, reading("STALZONE"))
	slider, _ := o.child(t, func(c window.Control) bool { return c.Kind == window.Slider && c.Knob.Name == "gap" })
	procSetFocus.Call(name)
	words, _ := windows.UTF16PtrFromString("STAL")
	procSetWindowTextW.Call(name, uintptr(unsafe.Pointer(words)))

	procSendMessageW.Call(slider, wmKeyDown, vkRight, 0)

	if got := textOf(name); got != "STAL" {
		t.Errorf("the name field reads %q while the player types, want STAL", got)
	}
}

func TestAClickOnASwatchTakesItsColour(t *testing.T) {
	o := open(t, listed())
	swatch, _ := o.child(t, func(c window.Control) bool { return c.Kind == window.Swatch && c.Colour == (crosshair.RGB{R: 255}) })

	procSendMessageW.Call(swatch, bmClick, 0, 0)

	if got := first(t, o.state.Current()).Crosshair.Color; got != (crosshair.RGB{R: 255}) {
		t.Errorf("colour %v after the click, want red", got)
	}
}
