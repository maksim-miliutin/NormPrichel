package window

import (
	"image"
	"image/color"
	"math"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

type Plate struct {
	Size   image.Point
	Radius int // pixels; half the height makes a pill
	Fill   crosshair.RGB
	Edge   crosshair.RGB
	Width  int // pixels of edge; zero lets the fill reach the border
	Back   crosshair.RGB
}

// GDI rounds corners in hard steps; a plate shades each pixel by how much of it the shape covers.
func (p Plate) Image() *image.RGBA {
	return p.paint(func(int, int) crosshair.RGB { return p.Fill })
}

func (p Plate) Around(picture *image.RGBA) *image.RGBA {
	p.Size = picture.Rect.Size()
	origin := picture.Rect.Min

	return p.paint(func(x, y int) crosshair.RGB {
		c := picture.RGBAAt(origin.X+x, origin.Y+y)
		return crosshair.RGB{R: c.R, G: c.G, B: c.B}
	})
}

func (p Plate) paint(fill func(x, y int) crosshair.RGB) *image.RGBA {
	img := image.NewRGBA(image.Rectangle{Max: p.Size})
	radius := min(float64(p.Radius), float64(p.Size.X)/2, float64(p.Size.Y)/2)
	edge := float64(p.Width)
	for y := range p.Size.Y {
		for x := range p.Size.X {
			cx, cy := float64(x)+0.5, float64(y)+0.5
			outer := coverage(cx, cy, image.Rect(0, 0, p.Size.X, p.Size.Y), radius)
			core := coverage(cx, cy, image.Rect(p.Width, p.Width, p.Size.X-p.Width, p.Size.Y-p.Width), max(radius-edge, 0))
			img.SetRGBA(x, y, mix(p.Back, p.Edge, fill(x, y), outer, core))
		}
	}

	return img
}

// coverage is the share of the pixel centred at (x, y) that lies inside the rounded rectangle.
func coverage(x, y float64, r image.Rectangle, radius float64) float64 {
	halfW, halfH := float64(r.Dx())/2, float64(r.Dy())/2
	qx := math.Abs(x-float64(r.Min.X)-halfW) - (halfW - radius)
	qy := math.Abs(y-float64(r.Min.Y)-halfH) - (halfH - radius)
	distance := math.Hypot(max(qx, 0), max(qy, 0)) + min(max(qx, qy), 0) - radius

	return min(max(0.5-distance, 0), 1)
}

func mix(back, edge, fill crosshair.RGB, outer, core float64) color.RGBA {
	channel := func(b, e, f uint8) uint8 {
		return uint8(math.Round(float64(b)*(1-outer) + float64(e)*(outer-core) + float64(f)*core))
	}

	return color.RGBA{R: channel(back.R, edge.R, fill.R), G: channel(back.G, edge.G, fill.G), B: channel(back.B, edge.B, fill.B), A: 0xFF}
}
