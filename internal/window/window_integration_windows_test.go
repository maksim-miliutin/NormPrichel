//go:build integration

package window_test

import (
	"fmt"
	"image"
	"maps"
	"runtime"
	"slices"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
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
	procRedrawWindow      = user32.NewProc("RedrawWindow")
	procGetClientRect     = user32.NewProc("GetClientRect")
	procGetDC             = user32.NewProc("GetDC")
	procReleaseDC         = user32.NewProc("ReleaseDC")
	procGetPixel          = windows.NewLazySystemDLL("gdi32.dll").NewProc("GetPixel")
	procGetDpiForWindow   = user32.NewProc("GetDpiForWindow")
	procGetFocus          = user32.NewProc("GetFocus")
	procSelectObject      = windows.NewLazySystemDLL("gdi32.dll").NewProc("SelectObject")
	procGetTextFaceW      = windows.NewLazySystemDLL("gdi32.dll").NewProc("GetTextFaceW")
	procGetObjectW        = windows.NewLazySystemDLL("gdi32.dll").NewProc("GetObjectW")

	procDwmGetWindowAttribute = windows.NewLazySystemDLL("dwmapi.dll").NewProc("DwmGetWindowAttribute")
)

const (
	idOK            = 1
	idCancel        = 2
	idiApplication  = 32512
	wmGetIcon       = 0x007F
	bnDoubleClicked = 5
	wmClose         = 0x0010
	wmKeyDown       = 0x0100
	wmLButtonDown   = 0x0201
	wmCommand       = 0x0111
	bmClick         = 0x00F5
	lbSetCurSel     = 0x0186
	enKillFocus     = 0x0200
	lbnSelChange    = 1
	vkRight         = 0x27
	vkControl       = 0x11
	vkMenu          = 0x12
	rdwInvalidate   = 0x0001
	rdwErase        = 0x0004
	rdwAllChildren  = 0x0080
	rdwUpdateNow    = 0x0100
	wmGetFont       = 0x0031
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

func clientOf(hwnd uintptr) image.Point {
	var client struct{ Left, Top, Right, Bottom int32 }
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))

	return image.Pt(int(client.Right), int(client.Bottom))
}

// pixel reads what a window shows at a point of its client area; a negative x counts from the right.
func pixel(hwnd uintptr, x, y int32) crosshair.RGB {
	if x < 0 {
		x += int32(clientOf(hwnd).X)
	}
	dc, _, _ := procGetDC.Call(hwnd)
	defer procReleaseDC.Call(hwnd, dc)
	ref, _, _ := procGetPixel.Call(dc, uintptr(x), uintptr(y))

	return crosshair.RGB{R: uint8(ref), G: uint8(ref >> 8), B: uint8(ref >> 16)}
}

func TestTheWindowIsDarkAndWhatIsChosenGlowsAmber(t *testing.T) {
	o := open(t, listed())
	o.actions.Form.AddGame(foreground.Choice{Title: "STALZONE", Exe: "STALZONE.EXE"})
	o.window.Show()
	procRedrawWindow.Call(o.window.Handle(), 0, 0, rdwInvalidate|rdwErase|rdwAllChildren|rdwUpdateNow)

	tab, _ := o.child(t, worded("readyMade"))
	button, _ := o.child(t, worded("addGame"))
	name, _ := o.child(t, func(c window.Control) bool { return c.Kind == window.Edit && c.Text == "STALZONE" })
	problem, _ := o.child(t, func(c window.Control) bool { return c.Kind == window.Problem })
	ground, chosen, idle, typed := pixel(o.window.Handle(), 4, 4), pixel(tab, 3, 3), pixel(button, 3, 3), pixel(name, -4, 3)
	refusal := pixel(problem, 3, 3)

	if brightness(ground) > 0x90 || brightness(ground) < 0x30 {
		t.Errorf("the window is %v, want it dark graphite rather than light or black", ground)
	}
	if refusal.R <= refusal.G || refusal.R <= refusal.B || brightness(refusal) <= brightness(ground) {
		t.Errorf("the refusal is %v, want it rust red", refusal)
	}
	if !amber(chosen) {
		t.Errorf("the chosen tab is %v, want it amber", chosen)
	}
	if brightness(idle) <= brightness(ground) || brightness(idle) > 0xA0 {
		t.Errorf("an idle button is %v on a window of %v, want it a dark shade lighter", idle, ground)
	}
	if typed != idle {
		t.Errorf("a field is %v and an idle button %v, want them alike", typed, idle)
	}
	if corner := pixel(button, 0, 0); corner != ground {
		t.Errorf("the corner of a button is %v, want the window's %v showing past its rounding", corner, ground)
	}
}

func TestTheTitleBarIsDarkWhereWindowsTellsIt(t *testing.T) {
	o := open(t, listed())

	var dark int32
	if failed, _, _ := procDwmGetWindowAttribute.Call(o.window.Handle(), 20, uintptr(unsafe.Pointer(&dark)), unsafe.Sizeof(dark)); failed != 0 {
		t.Skipf("this Windows does not tell whether a title bar is dark (%#x)", failed)
	}
	if dark == 0 {
		t.Error("the title bar stays light over a dark window")
	}
}

func TestEveryControlIsMadeOfItsKind(t *testing.T) {
	o := open(t, listed())
	classes := map[window.Kind]string{
		window.Button: "Button", window.Choice: "Button", window.Switch: "Button", window.Colour: "Button", window.Swatch: "Button",
		window.Edit: "Edit", window.Hotkey: "Edit", window.List: "ListBox", window.Label: "Static",
		window.Heading: "Static", window.Hint: "Static", window.Problem: "Static", window.Preview: "Static",
		window.Slider: "NormPrichelSlider", window.Empty: "Static",
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

func TestAFieldSitsInARoundFrameThatGlowsWhileItHasTheKeys(t *testing.T) {
	o := open(t, listed())
	o.window.Show()
	procRedrawWindow.Call(o.window.Handle(), 0, 0, rdwInvalidate|rdwErase|rdwAllChildren|rdwUpdateNow)
	name, id := o.child(t, func(c window.Control) bool { return c.Kind == window.Edit && c.Text == "STALZONE" })
	_, saving := o.child(t, func(c window.Control) bool { return c.Kind == window.Edit && c.Typed })
	button, _ := o.child(t, worded("addGame"))
	dpi, _, _ := procGetDpiForWindow.Call(o.window.Handle())
	frames := map[int]window.Rect{}
	for _, p := range window.Places() {
		frames[p.ID] = window.Scale(p.Rect, int(dpi))
	}
	frame := frames[id]
	// Only what the window itself marks for painting is painted, so a frame it forgot shows stale.
	edgeOf := func(r window.Rect) crosshair.RGB {
		procRedrawWindow.Call(o.window.Handle(), 0, 0, rdwAllChildren|rdwUpdateNow)
		return pixel(o.window.Handle(), int32(r.X), int32(r.Y+r.H/2))
	}
	edge := func() crosshair.RGB { return edgeOf(frame) }

	procSetFocus.Call(button)
	idle, ground := edge(), pixel(o.window.Handle(), 4, 4)
	procSetFocus.Call(name)
	focused := edge()
	corner := pixel(o.window.Handle(), int32(frame.X), int32(frame.Y))
	o.actions.Form.RemoveGame()
	o.actions.Form.RemoveGame()
	o.window.Show()
	gone := edgeOf(frames[saving])

	if idle == ground || amber(idle) {
		t.Errorf("the edge of an idle field is %v on a window of %v, want a quiet line", idle, ground)
	}
	if !amber(focused) {
		t.Errorf("the edge of the field with the keys is %v, want it amber", focused)
	}
	if corner != ground {
		t.Errorf("the corner of the frame is %v, want the window's %v past the rounding", corner, ground)
	}
	if gone != ground {
		t.Errorf("with no game left the frame of the hidden field to save under still shows %v", gone)
	}
}

func TestAClickOnAFieldsFrameGivesTheFieldTheKeys(t *testing.T) {
	o := open(t, listed())
	o.window.Show()
	name, id := o.child(t, func(c window.Control) bool { return c.Kind == window.Edit && c.Text == "STALZONE" })
	button, _ := o.child(t, worded("addGame"))
	var frame window.Rect
	for _, p := range window.Places() {
		if p.ID == id {
			frame = p.Rect
		}
	}
	procSetFocus.Call(button)

	dpi, _, _ := procGetDpiForWindow.Call(o.window.Handle())
	at := window.Scale(window.Rect{X: frame.X + 2, Y: frame.Y + frame.H/2}, int(dpi))
	procSendMessageW.Call(o.window.Handle(), wmLButtonDown, 0, uintptr(uint16(at.X))|uintptr(uint16(at.Y))<<16)
	if focus, _, _ := procGetFocus.Call(); focus != name {
		t.Errorf("after a click on the frame the keys are with %#x, want the field %#x", focus, name)
	}
}

// differences lists the points where a window shows something other than the picture drawn for it.
func differences(hwnd uintptr, picture *image.RGBA, points ...image.Point) []string {
	var found []string
	for _, p := range points {
		if got, want := pixel(hwnd, int32(p.X), int32(p.Y)), at(picture, p.X, p.Y); got != want {
			found = append(found, fmt.Sprintf("%v shows %v, want %v", p, got, want))
		}
	}

	return found
}

func shownOf(o opened, kind window.Kind) []window.Control {
	var found []window.Control
	for _, c := range window.Controls(o.actions.Form.Screen(), o.actions.Form.Word) {
		if c.Kind == kind && c.Shown {
			found = append(found, c)
		}
	}

	return found
}

func TestASwitchShowsThePictureOfItsState(t *testing.T) {
	o := open(t, listed())
	o.window.Show()
	button, _ := o.child(t, worded("addGame"))
	procSetFocus.Call(button)
	procRedrawWindow.Call(o.window.Handle(), 0, 0, rdwInvalidate|rdwErase|rdwAllChildren|rdwUpdateNow)
	dpi, _, _ := procGetDpiForWindow.Call(o.window.Handle())

	states := map[bool]int{}
	for _, c := range shownOf(o, window.Switch) {
		hwnd, _, _ := procGetDlgItem.Call(o.window.Handle(), uintptr(c.ID))
		look := window.Toggle{Size: clientOf(hwnd), DPI: int(dpi), On: c.On}
		track, knob := look.Track(), look.Knob()
		middle := track.Min.Y + track.Dy()/2
		probes := []image.Point{
			{track.Min.X + 1, middle}, {track.Min.X + 3, middle}, {knob.Min.X - 2, middle},
			{knob.Min.X + knob.Dx()/2, middle}, {knob.Max.X + 1, middle}, {track.Max.X - 3, middle},
		}
		for _, wrong := range differences(hwnd, look.Image(), probes...) {
			t.Errorf("the switch %q, on %v: %s", c.Word, c.On, wrong)
		}
		states[c.On]++
	}
	if states[true] == 0 || states[false] == 0 {
		t.Errorf("the screen has %d switches on and %d off, want both kinds to look at", states[true], states[false])
	}
}

func TestASliderShowsThePictureOfItsValue(t *testing.T) {
	o := open(t, listed())
	o.window.Show()
	button, _ := o.child(t, worded("addGame"))
	procSetFocus.Call(button)
	procRedrawWindow.Call(o.window.Handle(), 0, 0, rdwInvalidate|rdwErase|rdwAllChildren|rdwUpdateNow)
	dpi, _, _ := procGetDpiForWindow.Call(o.window.Handle())

	sliders := shownOf(o, window.Slider)
	if len(sliders) == 0 {
		t.Fatal("no slider on the screen")
	}
	for _, c := range sliders {
		hwnd, _, _ := procGetDlgItem.Call(o.window.Handle(), uintptr(c.ID))
		size := clientOf(hwnd)
		track := window.NewTrack(size.X, int(dpi))
		left := track.ThumbAt(c.Knob.Knob, c.Knob.Value)
		centre, middle := left+track.Thumb/2, size.Y/2
		probes := []image.Point{
			{track.Thumb/2 + 2, middle}, {centre, middle}, {centre + track.Thumb/4, middle},
			{centre, middle - track.Thumb/2 + 1}, {size.X - track.Thumb/2 - 2, middle},
		}
		for _, wrong := range differences(hwnd, track.Image(size.Y, left, false), probes...) {
			t.Errorf("the slider %q at %d: %s", c.Knob.Name, c.Knob.Value, wrong)
		}
	}
}

func TestAnEmptySavedListShowsAQuietFrameThatTakesNoKeys(t *testing.T) {
	o := open(t, listed())
	o.window.Show()
	hint, id := o.child(t, worded("noSaved"))
	button, _ := o.child(t, worded("addGame"))
	procSetFocus.Call(button)
	procRedrawWindow.Call(o.window.Handle(), 0, 0, rdwInvalidate|rdwErase|rdwAllChildren|rdwUpdateNow)
	dpi, _, _ := procGetDpiForWindow.Call(o.window.Handle())
	var frame window.Rect
	for _, p := range window.Places() {
		if p.ID == id {
			frame = window.Scale(p.Rect, int(dpi))
		}
	}

	size := clientOf(hint)
	ground, edge := pixel(o.window.Handle(), 4, 4), pixel(o.window.Handle(), int32(frame.X), int32(frame.Y+frame.H/2))
	inside, idle := pixel(hint, int32(size.X-2), int32(size.Y-2)), pixel(button, 3, 3)
	if edge == ground || amber(edge) {
		t.Errorf("the edge of the empty list is %v on a window of %v, want a quiet line", edge, ground)
	}
	if inside != idle {
		t.Errorf("inside the empty list is %v, want the shade of an idle button %v", inside, idle)
	}

	procSendMessageW.Call(o.window.Handle(), wmLButtonDown, 0, uintptr(uint16(frame.X+2))|uintptr(uint16(frame.Y+frame.H/2))<<16)
	if focus, _, _ := procGetFocus.Call(); focus != button {
		t.Errorf("a click on the empty list moved the keys to %#x, want them left with %#x", focus, button)
	}
}

type logFont struct {
	Height, Width, Escapement, Orientation, Weight int32
	Italic, Underline, StrikeOut, CharSet          uint8
	OutPrecision, ClipPrecision, Quality, Pitch    uint8
	Face                                           [32]uint16
}

// faces tells the face a font asks for and the face Windows gave it when it was made.
func faces(font uintptr) (asked logFont, given string) {
	procGetObjectW.Call(font, unsafe.Sizeof(asked), uintptr(unsafe.Pointer(&asked)))
	dc, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, dc)
	previous, _, _ := procSelectObject.Call(dc, font)
	defer procSelectObject.Call(dc, previous)
	name := make([]uint16, 32)
	procGetTextFaceW.Call(dc, uintptr(len(name)), uintptr(unsafe.Pointer(&name[0])))

	return asked, windows.UTF16ToString(name)
}

func TestEveryControlWritesInAFaceWindowsHasAndHeadingsInSemibold(t *testing.T) {
	o := open(t, listed())
	known := map[string]bool{"Segoe UI": true, "Segoe UI Variable Text": true, "Segoe UI Variable Text Semibold": true}

	seen := map[string]bool{}
	for _, p := range window.Places() {
		if p.Kind == window.Slider {
			continue
		}
		hwnd, _, _ := procGetDlgItem.Call(o.window.Handle(), uintptr(p.ID))
		font, _, _ := procSendMessageW.Call(hwnd, wmGetFont, 0, 0)
		asked, given := faces(font)
		face := windows.UTF16ToString(asked.Face[:])
		seen[face] = true
		if !known[face] || face != "Segoe UI" && face != given {
			t.Errorf("control %d asks for %q and gets %q, want Segoe UI Variable only where Windows has it", p.ID, face, given)
		}
		if heading := p.Kind == window.Heading; heading != (asked.Weight >= 600) {
			t.Errorf("control %d of kind %s writes at weight %d", p.ID, p.Kind, asked.Weight)
		}
	}
	t.Logf("the window writes in %v", slices.Sorted(maps.Keys(seen)))
}

func TestTheClipboardKeepsACodeAndTheButtonsUseIt(t *testing.T) {
	o := open(t, listed())
	copyButton, _ := o.child(t, worded("copyCode"))
	pasteButton, _ := o.child(t, worded("pasteCode"))

	procSendMessageW.Call(copyButton, bmClick, 0, 0)
	if got, err := o.window.Paste(); err != nil || got != profiles.DefaultCrosshair().Code() {
		t.Errorf("after the copy the clipboard holds %q, %v; want the code of the crosshair on show", got, err)
	}

	pasted := crosshair.Default()
	pasted.Shape = crosshair.Diamond
	if err := o.window.Copy(pasted.Code()); err != nil {
		t.Fatal(err)
	}
	procSendMessageW.Call(pasteButton, bmClick, 0, 0)
	if got := first(t, o.state.Current()).Crosshair.Shape; got != crosshair.Diamond {
		t.Errorf("after the paste the crosshair is a %s, want the diamond from the clipboard", got)
	}
}
