package window

import (
	"image/color"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

var (
	procBeginPaint        = user32.NewProc("BeginPaint")
	procEndPaint          = user32.NewProc("EndPaint")
	procGetClientRect     = user32.NewProc("GetClientRect")
	procFillRect          = user32.NewProc("FillRect")
	procFrameRect         = user32.NewProc("FrameRect")
	procDrawTextW         = user32.NewProc("DrawTextW")
	procDrawFocusRect     = user32.NewProc("DrawFocusRect")
	procSetCapture        = user32.NewProc("SetCapture")
	procReleaseCapture    = user32.NewProc("ReleaseCapture")
	procGetCapture        = user32.NewProc("GetCapture")
	procCreateSolidBrush  = gdi32.NewProc("CreateSolidBrush")
	procSelectObject      = gdi32.NewProc("SelectObject")
	procSetBkMode         = gdi32.NewProc("SetBkMode")
	procSetDIBitsToDevice = gdi32.NewProc("SetDIBitsToDevice")
)

const (
	odtListbox = 2

	odsSelected    = 0x0001
	odsFocus       = 0x0010
	odsNoFocusRect = 0x0200

	dtLeft        = 0x0000
	dtCenter      = 0x0001
	dtVCenter     = 0x0004
	dtSingleLine  = 0x0020
	dtNoPrefix    = 0x0800
	dtEndEllipsis = 0x8000

	transparent = 1

	biRGB        = 0
	dibRGBColors = 0
)

type drawItem struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	Hwnd       uintptr
	DC         uintptr
	Rect       rect
	ItemData   uintptr
}

type paintStruct struct {
	DC        uintptr
	Erase     int32
	Paint     rect
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
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

type previews struct {
	side  int
	dark  []byte
	light []byte
}

func (w *Window) draw(item *drawItem) {
	c := w.shown[int(item.CtlID)]
	dc, r := item.DC, item.Rect
	procSetBkMode.Call(dc, transparent)
	previous, _, _ := procSelectObject.Call(dc, w.font)
	defer procSelectObject.Call(dc, previous)

	switch {
	case item.CtlType == odtListbox:
		w.drawRow(item, c)
	case c.Kind == Preview:
		w.drawPreview(dc, r, c)
	case c.Kind == Problem:
		fill(dc, r, black)
		inner := rect{r.Left + int32(scaled(8, w.dpi)), r.Top, r.Right, r.Bottom}
		text(dc, inner, c.Text, white, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
	case c.Kind == Switch:
		w.drawSwitch(item, c)
	case c.Kind == Colour || c.Kind == Swatch:
		w.drawSwatch(item, c)
	default:
		inverted := c.On || item.ItemState&odsSelected != 0
		ink, paper := uint32(black), uint32(white)
		if inverted {
			ink, paper = white, black
		}
		fill(dc, r, paper)
		frame(dc, r, black)
		if c.Shape != "" {
			w.drawIcon(dc, r, c.Shape, inverted)
		} else {
			text(dc, r, c.Text, ink, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
		}
		focusRing(dc, item, inset(r, int32(scaled(3, w.dpi))))
	}
}

type iconKey struct {
	shape    crosshair.Shape
	side     int
	inverted bool
}

func (w *Window) drawIcon(dc uintptr, r rect, shape crosshair.Shape, inverted bool) {
	side := int(min(r.Right-r.Left, r.Bottom-r.Top)) - 2*scaled(5, w.dpi)
	key := iconKey{shape: shape, side: side, inverted: inverted}
	pixels, ok := w.shapeIcons[key]
	if !ok && side > 0 {
		ink, paper := color.RGBA{A: 255}, color.RGBA{R: 255, G: 255, B: 255, A: 255}
		if inverted {
			ink, paper = paper, ink
		}
		icon, err := Icon(shape, side, ink, paper)
		if err != nil {
			return
		}
		pixels = bgra(icon)
		w.shapeIcons[key] = pixels
	}
	if len(pixels) == 0 {
		return
	}

	x, y := r.Left+(r.Right-r.Left-int32(side))/2, r.Top+(r.Bottom-r.Top-int32(side))/2
	header := bitmapInfoHeader{Width: int32(side), Height: -int32(side), Planes: 1, BitCount: 32, Compression: biRGB}
	header.Size = uint32(unsafe.Sizeof(header))
	procSetDIBitsToDevice.Call(dc, uintptr(x), uintptr(y), uintptr(side), uintptr(side), 0, 0, 0, uintptr(side),
		uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&header)), dibRGBColors)
}

func (w *Window) drawSwitch(item *drawItem, c Control) {
	dc, r := item.DC, item.Rect
	side := int32(scaled(14, w.dpi))
	box := rect{r.Left, r.Top + (r.Bottom-r.Top-side)/2, r.Left + side, r.Top + (r.Bottom-r.Top-side)/2 + side}

	fill(dc, r, white)
	frame(dc, box, black)
	if c.On {
		fill(dc, inset(box, int32(scaled(3, w.dpi))), black)
	}
	if item.ItemState&odsSelected != 0 {
		frame(dc, inset(box, 1), black)
	}
	words := rect{box.Right + int32(scaled(6, w.dpi)), r.Top, r.Right, r.Bottom}
	text(dc, words, c.Text, black, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
	focusRing(dc, item, rect{words.Left - 2, words.Top + 2, words.Right, words.Bottom - 2})
}

// The square of the player's own colour shows that colour only while it is in use; until then it
// offers the dialog with an ellipsis.
func (w *Window) drawSwatch(item *drawItem, c Control) {
	dc, r := item.DC, item.Rect
	offer := c.Kind == Colour && !c.On
	paper := colourRef(c.Colour)
	if offer {
		paper = white
	}

	fill(dc, r, paper)
	frame(dc, r, black)
	if offer {
		text(dc, r, c.Text, black, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
	}
	if c.On || item.ItemState&odsSelected != 0 {
		frame(dc, inset(r, 1), white)
		frame(dc, inset(r, 2), black)
	}
	focusRing(dc, item, inset(r, int32(scaled(4, w.dpi))))
}

func (w *Window) drawRow(item *drawItem, c Control) {
	dc, r := item.DC, item.Rect
	if int(item.ItemID) >= len(c.Rows) {
		focusRing(dc, item, r)
		return
	}

	ink, paper := uint32(black), uint32(white)
	if item.ItemState&odsSelected != 0 {
		ink, paper = white, black
	}
	fill(dc, r, paper)
	words := rect{r.Left + int32(scaled(6, w.dpi)), r.Top, r.Right - int32(scaled(4, w.dpi)), r.Bottom}
	text(dc, words, c.Rows[item.ItemID], ink, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
	focusRing(dc, item, r)
}

func (w *Window) drawPreview(dc uintptr, r rect, c Control) {
	side := int(r.Right - r.Left)
	if w.previews == nil || w.previews.side != side {
		w.previews = w.paintPreviews(side)
	}

	pixels := w.previews.light
	if c.ID == idPreviewDark {
		pixels = w.previews.dark
	}
	header := bitmapInfoHeader{Width: int32(side), Height: -int32(side), Planes: 1, BitCount: 32, Compression: biRGB}
	header.Size = uint32(unsafe.Sizeof(header))
	procSetDIBitsToDevice.Call(dc, uintptr(r.Left), uintptr(r.Top), uintptr(side), uintptr(side), 0, 0, 0, uintptr(side),
		uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&header)), dibRGBColors)
	frame(dc, r, black)
}

func (w *Window) paintPreviews(side int) *previews {
	dark, light, _ := w.actions.Form.Preview(side)

	return &previews{side: side, dark: bgra(dark), light: bgra(light)}
}

// Windows hides focus rectangles until the keyboard is used, and the item says when.
func focusRing(dc uintptr, item *drawItem, r rect) {
	if item.ItemState&odsFocus != 0 && item.ItemState&odsNoFocusRect == 0 {
		procDrawFocusRect.Call(dc, uintptr(unsafe.Pointer(&r)))
	}
}

func fill(dc uintptr, r rect, colour uint32) {
	brush, _, _ := procCreateSolidBrush.Call(uintptr(colour))
	procFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), brush)
	procDeleteObject.Call(brush)
}

func frame(dc uintptr, r rect, colour uint32) {
	brush, _, _ := procCreateSolidBrush.Call(uintptr(colour))
	procFrameRect.Call(dc, uintptr(unsafe.Pointer(&r)), brush)
	procDeleteObject.Call(brush)
}

func text(dc uintptr, r rect, words string, colour uint32, format uintptr) {
	procSetTextColor.Call(dc, uintptr(colour))
	line, _ := windows.UTF16FromString(words)
	procDrawTextW.Call(dc, uintptr(unsafe.Pointer(&line[0])), uintptr(len(line)-1), uintptr(unsafe.Pointer(&r)), format)
}

func inset(r rect, by int32) rect {
	return rect{r.Left + by, r.Top + by, r.Right - by, r.Bottom - by}
}

// A COLORREF holds red in its lowest byte.
func colourRef(rgb crosshair.RGB) uint32 {
	return uint32(rgb.R) | uint32(rgb.G)<<8 | uint32(rgb.B)<<16
}

func sliderProc(hwnd, message, wparam, lparam uintptr) uintptr {
	w := running
	id, ok := 0, false
	if w != nil {
		id, ok = w.ids[hwnd]
	}
	if !ok {
		result, _, _ := procDefWindowProcW.Call(hwnd, message, wparam, lparam)
		return result
	}

	knob := w.shown[id].Knob
	switch message {
	case wmGetDlgCode:
		return dlgcWantArrows
	case wmEraseBkgnd:
		return 1
	case wmPaint:
		w.paintSlider(hwnd, id)
		return 0
	case wmSetFocus, wmKillFocus:
		procInvalidateRect.Call(hwnd, 0, 0)
		return 0
	case wmLButtonDown:
		procSetFocus.Call(hwnd)
		procSetCapture.Call(hwnd)
		w.slideTo(hwnd, id, int(int16(lparam&0xFFFF)))
		return 0
	case wmMouseMove:
		if captured, _, _ := procGetCapture.Call(); captured == hwnd {
			w.slideTo(hwnd, id, int(int16(lparam&0xFFFF)))
		}
		return 0
	case wmLButtonUp:
		procReleaseCapture.Call()
		return 0
	case wmKeyDown:
		if value, taken := Keyed(knob.Knob, knob.Value, uint32(wparam)); taken {
			if value != knob.Value {
				w.act(w.actions.Slide(id, value), false)
			}
			return 0
		}
	}

	result, _, _ := procDefWindowProcW.Call(hwnd, message, wparam, lparam)

	return result
}

func (w *Window) track(hwnd uintptr) (Track, rect) {
	var client rect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))

	return Track{Width: int(client.Right), Thumb: scaled(10, w.dpi)}, client
}

func (w *Window) slideTo(hwnd uintptr, id, x int) {
	knob := w.shown[id].Knob
	track, _ := w.track(hwnd)
	if value := track.ValueAt(knob.Knob, x); value != knob.Value {
		w.act(w.actions.Slide(id, value), false)
	}
}

func (w *Window) paintSlider(hwnd uintptr, id int) {
	var paint paintStruct
	dc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))
	defer procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&paint)))

	knob := w.shown[id].Knob
	track, client := w.track(hwnd)
	fill(dc, client, white)

	line := max(int32(scaled(1, w.dpi)), 1)
	middle := client.Bottom / 2
	half := int32(track.Thumb / 2)
	fill(dc, rect{half, middle - line/2, client.Right - half, middle - line/2 + line}, black)

	left := int32(track.ThumbAt(knob.Knob, knob.Value))
	margin := int32(scaled(4, w.dpi))
	fill(dc, rect{left, margin, left + int32(track.Thumb), client.Bottom - margin}, black)

	focus, _, _ := procGetFocus.Call()
	state, _, _ := procSendMessageW.Call(hwnd, wmQueryUIState, 0, 0)
	if focus == hwnd && state&uisfHideFocus == 0 {
		procDrawFocusRect.Call(dc, uintptr(unsafe.Pointer(&client)))
	}
}
