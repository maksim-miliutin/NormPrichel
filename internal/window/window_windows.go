package window

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/settings"
)

var ErrWindow = errors.New("window: the settings window cannot be made")

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	comdlg32 = windows.NewLazySystemDLL("comdlg32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	uxtheme  = windows.NewLazySystemDLL("uxtheme.dll")

	procRegisterClassExW         = user32.NewProc("RegisterClassExW")
	procCreateWindowExW          = user32.NewProc("CreateWindowExW")
	procDefWindowProcW           = user32.NewProc("DefWindowProcW")
	procCallWindowProcW          = user32.NewProc("CallWindowProcW")
	procSetWindowLongPtrW        = user32.NewProc("SetWindowLongPtrW")
	procDestroyWindow            = user32.NewProc("DestroyWindow")
	procShowWindow               = user32.NewProc("ShowWindow")
	procIsIconic                 = user32.NewProc("IsIconic")
	procSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procMoveWindow               = user32.NewProc("MoveWindow")
	procGetDpiForWindow          = user32.NewProc("GetDpiForWindow")
	procAdjustWindowRectExForDpi = user32.NewProc("AdjustWindowRectExForDpi")
	procMonitorFromWindow        = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
	procGetCursorPos             = user32.NewProc("GetCursorPos")
	procSendMessageW             = user32.NewProc("SendMessageW")
	procSetWindowTextW           = user32.NewProc("SetWindowTextW")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procInvalidateRect           = user32.NewProc("InvalidateRect")
	procGetFocus                 = user32.NewProc("GetFocus")
	procSetFocus                 = user32.NewProc("SetFocus")
	procIsChild                  = user32.NewProc("IsChild")
	procGetNextDlgTabItem        = user32.NewProc("GetNextDlgTabItem")
	procGetKeyState              = user32.NewProc("GetKeyState")
	procLoadCursorW              = user32.NewProc("LoadCursorW")
	procGetSystemMetricsForDpi   = user32.NewProc("GetSystemMetricsForDpi")
	procDestroyIcon              = user32.NewProc("DestroyIcon")
	procGetWindowRect            = user32.NewProc("GetWindowRect")
	procCreateFontW              = gdi32.NewProc("CreateFontW")
	procGetTextFaceW             = gdi32.NewProc("GetTextFaceW")
	procGetDC                    = user32.NewProc("GetDC")
	procReleaseDC                = user32.NewProc("ReleaseDC")
	procDeleteObject             = gdi32.NewProc("DeleteObject")
	procSetTextColor             = gdi32.NewProc("SetTextColor")
	procSetBkColor               = gdi32.NewProc("SetBkColor")
	procDwmSetWindowAttribute    = dwmapi.NewProc("DwmSetWindowAttribute")
	procSetWindowTheme           = uxtheme.NewProc("SetWindowTheme")
)

const (
	wsChild        = 0x40000000
	wsOverlapped   = 0x00000000
	wsCaption      = 0x00C00000
	wsSysMenu      = 0x00080000
	wsMinimizeBox  = 0x00020000
	wsClipChildren = 0x02000000
	wsTabStop      = 0x00010000
	wsGroup        = 0x00020000
	wsVScroll      = 0x00200000

	bsOwnerDraw         = 0x0000000B
	esAutoHScroll       = 0x0080
	ssOwnerDraw         = 0x0000000D
	ssNoPrefix          = 0x00000080
	lbsNotify           = 0x0001
	lbsOwnerDrawFixed   = 0x0010
	lbsHasStrings       = 0x0040
	lbsNoIntegralHeight = 0x0100

	wmActivate        = 0x0006
	wmSetFocus        = 0x0007
	wmKillFocus       = 0x0008
	wmPaint           = 0x000F
	wmClose           = 0x0010
	wmEraseBkgnd      = 0x0014
	wmSetFont         = 0x0030
	wmDrawItem        = 0x002B
	wmMeasureItem     = 0x002C
	wmSetIcon         = 0x0080
	wmGetDlgCode      = 0x0087
	wmKeyDown         = 0x0100
	wmChar            = 0x0102
	wmSysKeyDown      = 0x0104
	wmSysChar         = 0x0106
	wmCommand         = 0x0111
	wmCtlColorEdit    = 0x0133
	wmCtlColorListbox = 0x0134
	wmCtlColorStatic  = 0x0138
	wmMouseMove       = 0x0200
	wmLButtonDown     = 0x0201
	wmLButtonUp       = 0x0202
	wmQueryUIState    = 0x0129
	wmDpiChanged      = 0x02E0

	waInactive = 0

	bnClicked       = 0
	bnDoubleClicked = 5
	enSetFocus      = 0x0100
	enKillFocus     = 0x0200
	lbnSelChange    = 1
	lbnDblClk       = 2
	lbnSetFocus     = 4
	lbnKillFocus    = 5

	emSetLimitText  = 0x00C5
	lbAddString     = 0x0180
	lbResetContent  = 0x0184
	lbSetCurSel     = 0x0186
	lbGetCurSel     = 0x0188
	lbSetItemHeight = 0x01A0

	dlgcWantArrows  = 0x0001
	dlgcWantAllKeys = 0x0004

	idOK     = 1
	idCancel = 2

	swHide    = 0
	swShow    = 5
	swShowNA  = 8
	swRestore = 9

	swpNoZOrder   = 0x0004
	swpNoActivate = 0x0010

	monitorDefaultToNearest = 2

	fwNormal         = 400
	fwSemiBold       = 600
	defaultCharset   = 1
	clearTypeQuality = 5

	vkShift   = 0x10
	vkControl = 0x11
	vkMenu    = 0x12
	vkLWin    = 0x5B
	vkRWin    = 0x5C

	idcArrow = 32512

	gwlpWndProc = ^uintptr(3) // GWLP_WNDPROC is -4

	uisfHideFocus = 0x1

	iconSmall  = 0
	iconBig    = 1
	smCxIcon   = 11
	smCxSmIcon = 49
)

const windowStyle = wsOverlapped | wsCaption | wsSysMenu | wsMinimizeBox | wsClipChildren

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

type rect struct {
	Left, Top, Right, Bottom int32
}

type point struct {
	X, Y int32
}

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

type measureItem struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemWidth  uint32
	ItemHeight uint32
	ItemData   uintptr
}

// One settings window per process: its procedures are plain functions handed to Windows once,
// and they reach the window through running.
var (
	procedures = sync.OnceValue(func() callbacks {
		return callbacks{windows.NewCallback(windowProc), windows.NewCallback(sliderProc), windows.NewCallback(hotkeyProc)}
	})
	running *Window

	className, _       = windows.UTF16PtrFromString("NormPrichelSettings")
	sliderClassName, _ = windows.UTF16PtrFromString("NormPrichelSlider")
	title, _           = windows.UTF16PtrFromString("NormPrichel")
)

type callbacks struct {
	window, slider, hotkey uintptr
}

type Window struct {
	actions Actions
	changed func()
	fail    func(error)

	hwnd       uintptr
	dpi        int
	font       uintptr
	bold       uintptr
	children   map[int]uintptr
	ids        map[uintptr]int
	places     map[int]Place
	shown      map[int]Control
	ownProcs   map[uintptr]uintptr // the hotkey fields' own procedures, under ours
	focus      uintptr             // the control that had the keys when the window went to the back
	previews   *previews
	shapeIcons map[iconKey][]byte
	plates     map[Plate][]byte
	custom     [16]uint32 // the colour dialog's own colours, kept for the run
	icons      []uintptr
	brushes    map[crosshair.RGB]uintptr
	closing    bool
	apartment  bool
}

func New(form *settings.Form, changed func(), fail func(error)) (*Window, error) {
	instance, err := register()
	if err != nil {
		return nil, err
	}

	// The file dialog and the link go through COM, which wants the thread that shows them in a
	// single-threaded apartment.
	apartment := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE) == nil

	w := &Window{
		apartment:  apartment,
		changed:    changed,
		fail:       fail,
		children:   map[int]uintptr{},
		ids:        map[uintptr]int{},
		places:     map[int]Place{},
		shown:      map[int]Control{},
		ownProcs:   map[uintptr]uintptr{},
		shapeIcons: map[iconKey][]byte{},
		plates:     map[Plate][]byte{},
		brushes:    map[crosshair.RGB]uintptr{},
	}
	w.actions = Actions{Form: form, Inputs: w}
	running = w

	// Born under the cursor, the window takes the scale of the monitor the player is looking at.
	var at point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&at)))
	hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(title)),
		windowStyle, uintptr(at.X), uintptr(at.Y), 100, 100, 0, 0, uintptr(instance), 0)
	if hwnd == 0 {
		running = nil
		return nil, fmt.Errorf("%w: %w", ErrWindow, err)
	}
	w.hwnd = hwnd
	darkTitle(hwnd)
	dpi, _, _ := procGetDpiForWindow.Call(hwnd)
	w.dpi = max(int(dpi), 96)
	w.font, w.bold = makeFont(w.dpi, fwNormal), makeFont(w.dpi, fwSemiBold)

	for _, p := range Places() {
		if err := w.makeChild(p, instance); err != nil {
			w.Close()
			return nil, err
		}
	}
	w.centre()
	w.refresh(true)

	return w, nil
}

func register() (windows.Handle, error) {
	var instance windows.Handle
	if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
		return 0, fmt.Errorf("%w: %w", ErrWindow, err)
	}

	arrow, _, _ := procLoadCursorW.Call(0, idcArrow)
	for _, class := range []wndClassEx{
		{WndProc: procedures().window, Instance: instance, Cursor: windows.Handle(arrow), ClassName: className},
		{WndProc: procedures().slider, Instance: instance, Cursor: windows.Handle(arrow), ClassName: sliderClassName},
	} {
		class.Size = uint32(unsafe.Sizeof(class))
		atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&class)))
		if atom == 0 && !errors.Is(err, windows.ERROR_CLASS_ALREADY_EXISTS) {
			return 0, fmt.Errorf("%w: %w", ErrWindow, err)
		}
	}

	return instance, nil
}

func (w *Window) Handle() uintptr {
	return w.hwnd
}

func (w *Window) Show() {
	command := uintptr(swShow)
	if iconic, _, _ := procIsIconic.Call(w.hwnd); iconic != 0 {
		command = swRestore
	}
	procShowWindow.Call(w.hwnd, command)
	procSetForegroundWindow.Call(w.hwnd)
	w.refresh(false)
}

func (w *Window) Close() {
	w.closing = true
	procDestroyWindow.Call(w.hwnd)
	procDeleteObject.Call(w.font)
	procDeleteObject.Call(w.bold)
	for _, brush := range w.brushes {
		procDeleteObject.Call(brush)
	}
	for _, icon := range w.icons {
		procDestroyIcon.Call(icon)
	}
	if w.apartment {
		windows.CoUninitialize()
	}
	running = nil
}

func (w *Window) SetIcon(newIcon func(side int) (uintptr, error)) error {
	for _, size := range []struct{ which, metric uintptr }{{iconSmall, smCxSmIcon}, {iconBig, smCxIcon}} {
		side, _, _ := procGetSystemMetricsForDpi.Call(size.metric, uintptr(w.dpi))
		icon, err := newIcon(int(side))
		if err != nil {
			return err
		}
		w.icons = append(w.icons, icon)
		procSendMessageW.Call(w.hwnd, wmSetIcon, size.which, icon)
	}

	return nil
}

func (w *Window) makeChild(p Place, instance windows.Handle) error {
	class, style := "STATIC", uintptr(ssNoPrefix)
	switch p.Kind {
	case Button, Choice, Switch, Colour, Swatch:
		class, style = "BUTTON", bsOwnerDraw|wsTabStop|wsGroup
	case Edit, Hotkey:
		class, style = "EDIT", esAutoHScroll|wsTabStop|wsGroup
	case List:
		class, style = "LISTBOX", wsVScroll|lbsNotify|lbsOwnerDrawFixed|lbsHasStrings|lbsNoIntegralHeight|wsTabStop|wsGroup
	case Problem, Preview:
		style = ssOwnerDraw
	case Slider:
		class, style = "NormPrichelSlider", wsTabStop|wsGroup
	}

	name, _ := windows.UTF16PtrFromString(class)
	r := Scale(Interior(p), w.dpi)
	hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(name)), 0, wsChild|style,
		uintptr(r.X), uintptr(r.Y), uintptr(r.W), uintptr(r.H), w.hwnd, uintptr(p.ID), uintptr(instance), 0)
	if hwnd == 0 {
		return fmt.Errorf("%w: control %d: %w", ErrWindow, p.ID, err)
	}

	w.children[p.ID], w.ids[hwnd], w.places[p.ID] = hwnd, p.ID, p
	procSendMessageW.Call(hwnd, wmSetFont, w.fontOf(p.Kind), 0)
	if limit, ok := limits[p.ID]; ok {
		procSendMessageW.Call(hwnd, emSetLimitText, limit, 0)
	}
	if p.Kind == Hotkey {
		own, _, _ := procSetWindowLongPtrW.Call(hwnd, gwlpWndProc, procedures().hotkey)
		w.ownProcs[hwnd] = own
	}
	if p.Kind == List && procSetWindowTheme.Find() == nil {
		// The theme Explorer itself uses in dark mode; it darkens the list's scroll bar.
		dark, _ := windows.UTF16PtrFromString("DarkMode_Explorer")
		procSetWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(dark)), 0)
	}

	return nil
}

func framed(kind Kind) bool {
	return kind == Edit || kind == Hotkey || kind == List || kind == Empty
}

// Fields and lists have no border of their own: the window paints a round frame around each,
// amber while the field has the keys.
func (w *Window) paintFrames() {
	var paint paintStruct
	dc, _, _ := procBeginPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&paint)))
	defer procEndPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&paint)))

	focus, _, _ := procGetFocus.Call()
	for id, p := range w.places {
		c := w.shown[id]
		if !framed(p.Kind) || !c.Shown {
			continue
		}
		plate := Plate{Radius: scaled(4, w.dpi), Fill: field, Edge: seam, Width: 1, Back: ground}
		if p.Kind == Hotkey && c.On {
			plate.Fill = rust
		}
		if focus == w.children[id] {
			plate.Edge, plate.Width = signal, scaled(2, w.dpi)
		}
		w.plate(dc, w.frameOf(p), plate)
	}
}

func (w *Window) frameOf(p Place) rect {
	r := Scale(p.Rect, w.dpi)

	return rect{int32(r.X), int32(r.Y), int32(r.X + r.W), int32(r.Y + r.H)}
}

func (w *Window) reframe(id int) {
	r := w.frameOf(w.places[id])
	procInvalidateRect.Call(w.hwnd, uintptr(unsafe.Pointer(&r)), 1)
}

// A click on the margin of a frame lands on the window, not the field, so it is passed on.
func (w *Window) focusFieldAt(x, y int) {
	for id, p := range w.places {
		r := Scale(p.Rect, w.dpi)
		takes := framed(p.Kind) && p.Kind != Empty && w.shown[id].Shown
		if takes && x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			procSetFocus.Call(w.children[id])
			return
		}
	}
}

// Characters a field takes: a game's name, the name to save under, an offset.
var limits = map[int]uintptr{idName: 80, idSaveName: 60, idX: 8, idY: 8}

func (w *Window) fontOf(kind Kind) uintptr {
	if kind == Heading {
		return w.bold
	}

	return w.font
}

var variableFaces = map[uintptr]string{fwNormal: "Segoe UI Variable Text", fwSemiBold: "Segoe UI Variable Text Semibold"}

// Windows 11 draws its own windows in Segoe UI Variable, which older Windows lacks; a missing face
// is swapped silently for another, so the font is kept only if it came in the face it asked for.
func makeFont(dpi int, weight uintptr) uintptr {
	face := variableFaces[weight]
	font := fontIn(face, dpi, weight)
	if faceOf(font) == face {
		return font
	}
	procDeleteObject.Call(font)

	return fontIn("Segoe UI", dpi, weight)
}

// A negative height asks for the size of the letters rather than of the line: 9 points.
func fontIn(face string, dpi int, weight uintptr) uintptr {
	name, _ := windows.UTF16PtrFromString(face)
	height := -int32((9*dpi + 36) / 72)
	font, _, _ := procCreateFontW.Call(uintptr(height), 0, 0, 0, weight, 0, 0, 0, defaultCharset, 0, 0, clearTypeQuality, 0, uintptr(unsafe.Pointer(name)))

	return font
}

func faceOf(font uintptr) string {
	dc, _, _ := procGetDC.Call(0)
	defer procReleaseDC.Call(0, dc)
	previous, _, _ := procSelectObject.Call(dc, font)
	defer procSelectObject.Call(dc, previous)
	name := make([]uint16, 32)
	procGetTextFaceW.Call(dc, uintptr(len(name)), uintptr(unsafe.Pointer(&name[0])))

	return windows.UTF16ToString(name)
}

func (w *Window) centre() {
	width, height := Size()
	client := Scale(Rect{W: width, H: height}, w.dpi)
	frame := rect{Right: int32(client.W), Bottom: int32(client.H)}
	procAdjustWindowRectExForDpi.Call(uintptr(unsafe.Pointer(&frame)), windowStyle, 0, 0, uintptr(w.dpi))

	monitor, _, _ := procMonitorFromWindow.Call(w.hwnd, monitorDefaultToNearest)
	info := monitorInfo{}
	info.Size = uint32(unsafe.Sizeof(info))
	procGetMonitorInfoW.Call(monitor, uintptr(unsafe.Pointer(&info)))

	outerW, outerH := frame.Right-frame.Left, frame.Bottom-frame.Top
	x := info.Work.Left + max((info.Work.Right-info.Work.Left-outerW)/2, 0)
	y := info.Work.Top + max((info.Work.Bottom-info.Work.Top-outerH)/2, 0)
	procSetWindowPos.Call(w.hwnd, 0, uintptr(x), uintptr(y), uintptr(outerW), uintptr(outerH), swpNoZOrder|swpNoActivate)
}

// A field the player is typing in keeps its words, unless the change came from that very field.
func (w *Window) refresh(everyField bool) {
	focus, _, _ := procGetFocus.Call()
	for _, c := range Controls(w.actions.Form.Screen(), w.actions.Form.Word) {
		hwnd := w.children[c.ID]
		old, seen := w.shown[c.ID]
		w.shown[c.ID] = c

		switch c.Kind {
		case Edit, Hotkey:
			if !c.Typed && (everyField || hwnd != focus) && textOf(hwnd) != c.Text {
				setText(hwnd, c.Text)
			}
		case List:
			w.fillList(hwnd, old, c, seen)
		default:
			if !seen || old.Text != c.Text {
				setText(hwnd, c.Text)
			}
		}
		if c.Kind == Preview || old.On != c.On || old.Colour != c.Colour || old.Knob != c.Knob || old.Text != c.Text {
			procInvalidateRect.Call(hwnd, 0, 1)
		}
		if !seen || old.Shown != c.Shown {
			procShowWindow.Call(hwnd, showing(c.Shown))
		}
		if framed(c.Kind) && (!seen || old.Shown != c.Shown || old.On != c.On) {
			w.reframe(c.ID)
		}
	}
	w.previews = nil

	if focus != 0 && w.ids[focus] != 0 && !w.shown[w.ids[focus]].Shown {
		next, _, _ := procGetNextDlgTabItem.Call(w.hwnd, 0, 0)
		procSetFocus.Call(next)
	}
}

func showing(shown bool) uintptr {
	if shown {
		return swShowNA
	}

	return swHide
}

func (w *Window) fillList(hwnd uintptr, old, c Control, seen bool) {
	if !seen || !slices.Equal(old.Rows, c.Rows) {
		procSendMessageW.Call(hwnd, lbResetContent, 0, 0)
		for _, row := range c.Rows {
			text, _ := windows.UTF16PtrFromString(row)
			procSendMessageW.Call(hwnd, lbAddString, 0, uintptr(unsafe.Pointer(text)))
		}
	}
	if c.Row >= 0 {
		procSendMessageW.Call(hwnd, lbSetCurSel, uintptr(c.Row), 0)
	}
}

func (w *Window) rescale(dpi int, suggested *rect) {
	w.dpi = dpi
	old := []uintptr{w.font, w.bold}
	w.font, w.bold = makeFont(dpi, fwNormal), makeFont(dpi, fwSemiBold)
	for id, hwnd := range w.children {
		p := w.places[id]
		r := Scale(Interior(p), dpi)
		procMoveWindow.Call(hwnd, uintptr(r.X), uintptr(r.Y), uintptr(r.W), uintptr(r.H), 1)
		procSendMessageW.Call(hwnd, wmSetFont, w.fontOf(p.Kind), 1)
		if p.Kind == List {
			procSendMessageW.Call(hwnd, lbSetItemHeight, 0, uintptr(scaled(20, dpi)))
		}
	}
	for _, font := range old {
		procDeleteObject.Call(font)
	}
	procSetWindowPos.Call(w.hwnd, 0, uintptr(suggested.Left), uintptr(suggested.Top),
		uintptr(suggested.Right-suggested.Left), uintptr(suggested.Bottom-suggested.Top), swpNoZOrder|swpNoActivate)
	w.previews = nil
	procInvalidateRect.Call(w.hwnd, 0, 1)
}

func (w *Window) act(err error, everyField bool) {
	if err != nil {
		w.fail(err)
	}
	w.refresh(everyField)
	w.changed()
}

func (w *Window) command(id, code int, child uintptr) {
	if w.closing {
		return
	}
	if child == 0 {
		w.dialogKey(id)
		return
	}

	switch w.places[id].Kind {
	case Button, Choice, Switch, Colour, Swatch:
		// A second click soon after the first comes as a double click, and the arrows would
		// move the crosshair at half the rate the player clicks.
		if code == bnClicked || code == bnDoubleClicked {
			w.act(w.actions.Click(id), false)
		}
	case Edit, Hotkey:
		switch code {
		case enSetFocus:
			w.reframe(id)
		case enKillFocus:
			w.reframe(id)
			w.act(w.actions.Leave(id), false)
		}
	case List:
		switch code {
		case lbnSelChange:
			w.act(w.actions.Select(id), false)
		case lbnDblClk:
			w.act(w.actions.Open(id), false)
		case lbnSetFocus, lbnKillFocus:
			w.reframe(id)
		}
	}
}

// Enter and Esc come as the dialog's OK and Cancel: Enter finishes the field or presses the
// button that has the keys, and Esc puts the window away.
func (w *Window) dialogKey(id int) {
	focus, _, _ := procGetFocus.Call()
	focused := w.ids[focus]
	switch {
	case id == idCancel:
		procShowWindow.Call(w.hwnd, swHide)
	case id == idOK && (w.places[focused].Kind == Edit || w.places[focused].Kind == Hotkey):
		w.act(w.actions.Enter(focused), true)
	case id == idOK && focused != 0:
		w.command(focused, bnClicked, focus)
	}
}

func (w *Window) activate(active bool) {
	if !active {
		focus, _, _ := procGetFocus.Call()
		if child, _, _ := procIsChild.Call(w.hwnd, focus); child != 0 {
			w.focus = focus
		}
		return
	}

	w.refresh(false)
	if w.focus != 0 && w.shown[w.ids[w.focus]].Shown {
		procSetFocus.Call(w.focus)
		return
	}
	first, _, _ := procGetNextDlgTabItem.Call(w.hwnd, 0, 0)
	procSetFocus.Call(first)
}

func windowProc(hwnd, message, wparam, lparam uintptr) uintptr {
	w := running
	if w == nil || w.hwnd != hwnd {
		result, _, _ := procDefWindowProcW.Call(hwnd, message, wparam, lparam)
		return result
	}

	switch message {
	case wmCommand:
		w.command(int(wparam&0xFFFF), int(wparam>>16&0xFFFF), lparam)
		return 0
	case wmDrawItem:
		w.draw(at[drawItem](lparam))
		return 1
	case wmMeasureItem:
		at[measureItem](lparam).ItemHeight = uint32(scaled(20, w.dpi))
		return 1
	case wmCtlColorStatic, wmCtlColorEdit, wmCtlColorListbox:
		return w.colours(wparam, lparam)
	case wmPaint:
		w.paintFrames()
		return 0
	case wmLButtonDown:
		w.focusFieldAt(int(int16(lparam&0xFFFF)), int(int16(lparam>>16&0xFFFF)))
		return 0
	case wmEraseBkgnd:
		var client rect
		procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
		procFillRect.Call(wparam, uintptr(unsafe.Pointer(&client)), w.brush(ground))
		return 1
	case wmActivate:
		w.activate(wparam&0xFFFF != waInactive)
		return 0
	case wmClose:
		procShowWindow.Call(hwnd, swHide)
		return 0
	case wmDpiChanged:
		w.rescale(int(wparam&0xFFFF), at[rect](lparam))
		return 0
	}

	result, _, _ := procDefWindowProcW.Call(hwnd, message, wparam, lparam)

	return result
}

// The message carries the address of a structure Windows owns. Read through the variable's own
// address, the conversion stays clear of the uintptr arithmetic vet guards against.
func at[T any](lparam uintptr) *T {
	return *(**T)(unsafe.Pointer(&lparam))
}

func (w *Window) colours(dc, child uintptr) uintptr {
	c := w.shown[w.ids[child]]
	words, paper := ink, ground
	switch {
	case c.Kind == Hint:
		words = ash
	case c.Kind == Empty:
		words, paper = ash, field
	case c.Kind == Hotkey && c.On:
		paper = rust
	case framed(c.Kind):
		paper = field
	}
	procSetTextColor.Call(dc, uintptr(colourRef(words)))
	procSetBkColor.Call(dc, uintptr(colourRef(paper)))

	return w.brush(paper)
}

// A control's colours come back to Windows as brushes it does not free, so each one is made once
// and lives as long as the window.
func (w *Window) brush(colour crosshair.RGB) uintptr {
	if brush, ok := w.brushes[colour]; ok {
		return brush
	}
	brush, _, _ := procCreateSolidBrush.Call(uintptr(colourRef(colour)))
	w.brushes[colour] = brush

	return brush
}

// Windows 10 before version 2004 knew the dark title bar as attribute 19, later ones as 20.
func darkTitle(hwnd uintptr) {
	if procDwmSetWindowAttribute.Find() != nil {
		return
	}
	on := int32(1)
	for _, attribute := range []uintptr{20, 19} {
		if failed, _, _ := procDwmSetWindowAttribute.Call(hwnd, attribute, uintptr(unsafe.Pointer(&on)), unsafe.Sizeof(on)); failed == 0 {
			return
		}
	}
}

// The hotkey fields take a pressed combination whole. While Ctrl, Alt or Win is held they ask
// IsDialogMessage for every key, Tab and Enter too, and swallow the characters the press makes.
func hotkeyProc(hwnd, message, wparam, lparam uintptr) uintptr {
	w := running
	own := w.ownProcs[hwnd]
	held := modifiersHeld()
	switch message {
	case wmGetDlgCode:
		if held&(modControl|modAlt|modWin) != 0 {
			code, _, _ := procCallWindowProcW.Call(own, hwnd, message, wparam, lparam)
			return code | dlgcWantAllKeys
		}
	case wmKeyDown, wmSysKeyDown:
		if held&(modControl|modAlt|modWin) == 0 {
			break
		}
		if taken, err := w.actions.Press(w.ids[hwnd], held, uint32(wparam)); taken {
			w.act(err, true)
			return 0
		}
	case wmChar, wmSysChar:
		if held&(modControl|modAlt|modWin) != 0 {
			return 0
		}
	}

	result, _, _ := procCallWindowProcW.Call(own, hwnd, message, wparam, lparam)

	return result
}

// The same flags as RegisterHotKey's.
const (
	modAlt     = 0x0001
	modControl = 0x0002
	modShift   = 0x0004
	modWin     = 0x0008
)

func modifiersHeld() uint32 {
	var held uint32
	for _, key := range []struct {
		code uintptr
		flag uint32
	}{{vkControl, modControl}, {vkMenu, modAlt}, {vkShift, modShift}, {vkLWin, modWin}, {vkRWin, modWin}} {
		if state, _, _ := procGetKeyState.Call(key.code); state&0x8000 != 0 {
			held |= key.flag
		}
	}

	return held
}

func textOf(hwnd uintptr) string {
	length, _, _ := procGetWindowTextLengthW.Call(hwnd)
	buffer := make([]uint16, length+1)
	procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))

	return windows.UTF16ToString(buffer)
}

func setText(hwnd uintptr, text string) {
	words, _ := windows.UTF16PtrFromString(text)
	procSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(words)))
}
