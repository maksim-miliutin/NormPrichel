package window

import (
	"image"
	"image/color"
	"math"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

const pill = 1 << 16

type Layer struct {
	Rect   image.Rectangle
	Radius int // pixels; past half the shorter side the ends come out as half circles
	Colour crosshair.RGB
}

// Unlike a plate, a layer shades its edge into whatever lies under it, so a thumb blends into its rail.
func Layered(size image.Point, back crosshair.RGB, layers ...Layer) *image.RGBA {
	img := image.NewRGBA(image.Rectangle{Max: size})
	for y := range size.Y {
		for x := range size.X {
			img.SetRGBA(x, y, shade(image.Pt(x, y), back, layers))
		}
	}

	return img
}

func shade(at image.Point, back crosshair.RGB, layers []Layer) color.RGBA {
	r, g, b := float64(back.R), float64(back.G), float64(back.B)
	for _, l := range layers {
		if !at.In(l.Rect) {
			continue
		}
		radius := min(float64(l.Radius), float64(l.Rect.Dx())/2, float64(l.Rect.Dy())/2)
		share := coverage(float64(at.X)+0.5, float64(at.Y)+0.5, l.Rect, radius)
		r, g, b = r+(float64(l.Colour.R)-r)*share, g+(float64(l.Colour.G)-g)*share, b+(float64(l.Colour.B)-b)*share
	}

	return color.RGBA{R: uint8(math.Round(r)), G: uint8(math.Round(g)), B: uint8(math.Round(b)), A: 0xFF}
}
