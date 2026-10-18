package window

import (
	"image"
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
		w.plate(dc, r, Plate{Radius: scaled(4, w.dpi), Fill: rust, Back: ground})
		inner := rect{r.Left + int32(scaled(8, w.dpi)), r.Top, r.Right, r.Bottom}
		text(dc, inner, c.Text, ink, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
	case c.Kind == Switch:
		w.drawSwitch(item, c)
	case c.Kind == Tab:
		w.drawTab(item, c)
	case c.Kind == Colour || c.Kind == Swatch:
		w.drawSwatch(item, c)
	default:
		chosen := c.On || item.ItemState&odsSelected != 0
		words, paper, edge := ink, field, seam
		if chosen {
			words, paper, edge = onSignal, signal, signal
		}
		w.plate(dc, r, w.focused(item, Plate{Radius: scaled(4, w.dpi), Fill: paper, Edge: edge, Width: 1, Back: ground}))
		if c.Shape != "" {
			w.drawIcon(dc, r, c.Shape, chosen)
		} else {
			text(dc, r, c.Text, words, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
		}
	}
}

func (w *Window) focused(item *drawItem, p Plate) Plate {
	if keyed(item) {
		p.Edge, p.Width = ink, scaled(2, w.dpi)
	}

	return p
}

// Windows keeps the marks of the keys hidden until the keyboard is used, and the item says when.
func keyed(item *drawItem) bool {
	return item.ItemState&odsFocus != 0 && item.ItemState&odsNoFocusRect == 0
}

func (w *Window) plate(dc uintptr, r rect, p Plate) {
	p.Size = image.Pt(int(r.Right-r.Left), int(r.Bottom-r.Top))
	pixels, ok := w.plates[p]
	if !ok {
		pixels = bgra(p.Image())
		w.plates[p] = pixels
	}
	blit(dc, r.Left, r.Top, p.Size, pixels)
}

func blit(dc uintptr, x, y int32, size image.Point, pixels []byte) {
	if len(pixels) == 0 {
		return
	}
	header := bitmapInfoHeader{Width: int32(size.X), Height: -int32(size.Y), Planes: 1, BitCount: 32, Compression: biRGB}
	header.Size = uint32(unsafe.Sizeof(header))
	procSetDIBitsToDevice.Call(dc, uintptr(x), uintptr(y), uintptr(size.X), uintptr(size.Y), 0, 0, 0, uintptr(size.Y),
		uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&header)), dibRGBColors)
}

type iconKey struct {
	shape  crosshair.Shape
	side   int
	chosen bool
}

func (w *Window) drawIcon(dc uintptr, r rect, shape crosshair.Shape, chosen bool) {
	side := int(min(r.Right-r.Left, r.Bottom-r.Top)) - 2*scaled(5, w.dpi)
	key := iconKey{shape: shape, side: side, chosen: chosen}
	pixels, ok := w.shapeIcons[key]
	if !ok && side > 0 {
		lines, paper := ink, field
		if chosen {
			lines, paper = onSignal, signal
		}
		icon, err := Icon(shape, side, opaque(lines), opaque(paper))
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
	blit(dc, x, y, image.Pt(side, side), pixels)
}

func (w *Window) drawSwitch(item *drawItem, c Control) {
	dc, r := item.DC, item.Rect
	look := Toggle{Size: image.Pt(int(r.Right-r.Left), int(r.Bottom-r.Top)), DPI: w.dpi, On: c.On, Pressed: item.ItemState&odsSelected != 0, Keyed: keyed(item)}
	blit(dc, r.Left, r.Top, look.Size, bgra(look.Image()))

	words := rect{r.Left + int32(look.Track().Max.X+scaled(8, w.dpi)), r.Top, r.Right, r.Bottom}
	text(dc, words, c.Text, ink, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
}

// Not a filled face like the choices on a page: the open tab is marked only by the line beneath it.
func (w *Window) drawTab(item *drawItem, c Control) {
	dc, r := item.DC, item.Rect
	line := int32(scaled(3, w.dpi))
	words, colour := rect{r.Left, r.Top, r.Right, r.Bottom - line}, ash
	fill(dc, r, ground)
	if c.On {
		fill(dc, rect{r.Left, r.Bottom - line, r.Right, r.Bottom}, signal)
		procSelectObject.Call(dc, w.bold)
		colour = ink
	}
	text(dc, words, c.Text, colour, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
	focusRing(dc, item, r)
}

// The square of the player's own colour shows that colour only while it is in use; until then it
// offers the dialog with an ellipsis.
func (w *Window) drawSwatch(item *drawItem, c Control) {
	dc, r := item.DC, item.Rect
	offer := c.Kind == Colour && !c.On
	p := Plate{Radius: scaled(4, w.dpi), Fill: c.Colour, Edge: seam, Width: 1, Back: ground}
	if offer {
		p.Fill = field
	}
	if c.On || item.ItemState&odsSelected != 0 {
		p.Edge, p.Width = signal, scaled(2, w.dpi)
	}

	w.plate(dc, r, w.focused(item, p))
	if offer {
		text(dc, r, c.Text, ink, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
	}
}

func (w *Window) drawRow(item *drawItem, c Control) {
	dc, r := item.DC, item.Rect
	if int(item.ItemID) >= len(c.Rows) {
		focusRing(dc, item, r)
		return
	}

	words := ink
	fill(dc, r, field)
	if item.ItemState&odsSelected != 0 {
		words = onSignal
		pill := rect{r.Left + int32(scaled(2, w.dpi)), r.Top + 1, r.Right - int32(scaled(2, w.dpi)), r.Bottom - 1}
		w.plate(dc, pill, Plate{Radius: scaled(4, w.dpi), Fill: signal, Back: field})
	}
	inner := rect{r.Left + int32(scaled(8, w.dpi)), r.Top, r.Right - int32(scaled(4, w.dpi)), r.Bottom}
	text(dc, inner, c.Rows[item.ItemID], words, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
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
	blit(dc, r.Left, r.Top, image.Pt(side, side), pixels)
}

func (w *Window) paintPreviews(side int) *previews {
	dark, light, _ := w.actions.Form.Preview(side)
	tile := Plate{Radius: scaled(6, w.dpi), Edge: seam, Width: 1, Back: ground}

	return &previews{side: side, dark: bgra(tile.Around(dark)), light: bgra(tile.Around(light))}
}

func focusRing(dc uintptr, item *drawItem, r rect) {
	if keyed(item) {
		procDrawFocusRect.Call(dc, uintptr(unsafe.Pointer(&r)))
	}
}

func fill(dc uintptr, r rect, colour crosshair.RGB) {
	brush, _, _ := procCreateSolidBrush.Call(uintptr(colourRef(colour)))
	procFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), brush)
	procDeleteObject.Call(brush)
}

func text(dc uintptr, r rect, words string, colour crosshair.RGB, format uintptr) {
	procSetTextColor.Call(dc, uintptr(colourRef(colour)))
	line, _ := windows.UTF16FromString(words)
	procDrawTextW.Call(dc, uintptr(unsafe.Pointer(&line[0])), uintptr(len(line)-1), uintptr(unsafe.Pointer(&r)), format)
}

// A COLORREF holds red in its lowest byte.
func colourRef(rgb crosshair.RGB) uint32 {
	return uint32(rgb.R) | uint32(rgb.G)<<8 | uint32(rgb.B)<<16
}

func opaque(rgb crosshair.RGB) color.RGBA {
	return color.RGBA{R: rgb.R, G: rgb.G, B: rgb.B, A: 0xFF}
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

	return NewTrack(int(client.Right), w.dpi), client
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
	focus, _, _ := procGetFocus.Call()
	state, _, _ := procSendMessageW.Call(hwnd, wmQueryUIState, 0, 0)
	picture := track.Image(int(client.Bottom), track.ThumbAt(knob.Knob, knob.Value), focus == hwnd && state&uisfHideFocus == 0)
	blit(dc, 0, 0, picture.Rect.Size(), bgra(picture))
}
