package window

import "image"

type Toggle struct {
	Size    image.Point
	DPI     int
	On      bool
	Pressed bool
	Keyed   bool // has the keys while the keyboard is in use
}

func (t Toggle) Track() image.Rectangle {
	corner := image.Pt(0, (t.Size.Y-scaled(18, t.DPI))/2)

	return image.Rectangle{Min: corner, Max: corner.Add(image.Pt(scaled(36, t.DPI), scaled(18, t.DPI)))}
}

// A pressed knob stretches toward the middle, as if it were about to slide over.
func (t Toggle) Knob() image.Rectangle {
	track := t.Track()
	pad := max(track.Dy()/5, 1)
	side := track.Dy() - 2*pad
	stretch := 0
	if t.Pressed {
		stretch = side / 4
	}
	top := track.Min.Y + pad
	if t.On {
		return image.Rect(track.Max.X-pad-side-stretch, top, track.Max.X-pad, top+side)
	}

	return image.Rect(track.Min.X+pad, top, track.Min.X+pad+side+stretch, top+side)
}

func (t Toggle) Image() *image.RGBA {
	edge, width, paper, knob := ash, 1, field, ash
	if t.On {
		edge, width, paper, knob = signal, 0, signal, onSignal
	}
	if t.Keyed {
		edge, width = ink, scaled(2, t.DPI)
	}
	track := t.Track()

	return Layered(t.Size, ground, Layer{track, pill, edge}, Layer{track.Inset(width), pill, paper}, Layer{t.Knob(), pill, knob})
}
