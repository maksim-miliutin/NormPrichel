package window

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

type ShapeIcon struct {
	Shape      crosshair.Shape
	Corners    crosshair.Corners
	Side       int
	Ink, Paper color.RGBA
}

func (i ShapeIcon) Image() (*image.RGBA, error) {
	side := i.Side
	drawn, err := crosshair.Draw(crosshair.Style{
		Shape:     i.Shape,
		Color:     crosshair.RGB{R: i.Ink.R, G: i.Ink.G, B: i.Ink.B},
		Opacity:   100,
		Length:    max(1, side/5),
		Thickness: max(1, side/16),
		Gap:       max(1, side/8),
		Radius:    max(2, side*3/10),
		DotSize:   max(2, side/10),
		Corners:   i.Corners,
	})
	if err != nil {
		return nil, err
	}

	out := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(out, out.Rect, &image.Uniform{C: i.Paper}, image.Point{}, draw.Src)
	// What the ink covers is centred, so a chevron or a tee sits in the middle of its button.
	inked := inkedBounds(drawn)
	at := image.Pt((side-inked.Dx())/2, (side-inked.Dy())/2).Sub(inked.Min)
	draw.Draw(out, drawn.Rect.Add(at), drawn, drawn.Rect.Min, draw.Over)

	return out, nil
}

func inkedBounds(img *image.RGBA) image.Rectangle {
	inked := image.Rectangle{}
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			if img.RGBAAt(x, y).A != 0 {
				inked = inked.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}

	return inked
}
