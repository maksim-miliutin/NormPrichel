package window

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

func Icon(shape crosshair.Shape, side int, ink, paper color.RGBA) (*image.RGBA, error) {
	thickness := max(1, side/16)
	drawn, err := crosshair.Draw(crosshair.Style{
		Shape:     shape,
		Color:     crosshair.RGB{R: ink.R, G: ink.G, B: ink.B},
		Opacity:   100,
		Length:    max(1, side/5),
		Thickness: thickness,
		Gap:       max(1, side/8),
		Radius:    max(2, side*3/10),
		DotSize:   max(2, side/10),
	})
	if err != nil {
		return nil, err
	}

	out := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(out, out.Rect, &image.Uniform{C: paper}, image.Point{}, draw.Src)
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
