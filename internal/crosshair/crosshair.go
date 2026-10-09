package crosshair

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"slices"
)

type Shape string

const (
	Dot      Shape = "dot"
	Cross    Shape = "cross"
	CrossDot Shape = "cross-dot"
	Tee      Shape = "t"
)

type RGB struct {
	R, G, B uint8
}

type Style struct {
	Shape     Shape
	Color     RGB
	Opacity   int // percent
	Length    int
	Thickness int
	Gap       int
	DotSize   int
}

var ErrUnknownShape = errors.New("crosshair: unknown shape")

var shapes = []struct {
	shape Shape
	parts func(s Style) parts
}{
	{Dot, func(s Style) parts { return withDot(parts{parity: s.DotSize % 2}, s) }},
	{Cross, func(s Style) parts { return withArms(s, true) }},
	{CrossDot, func(s Style) parts { return withDot(withArms(s, true), s) }},
	{Tee, func(s Style) parts { return withArms(s, false) }},
}

type limit struct {
	min, max int
}

var (
	opacityLimit   = limit{min: 10, max: 100}
	lengthLimit    = limit{min: 1, max: 64}
	thicknessLimit = limit{min: 1, max: 16}
	gapLimit       = limit{min: 0, max: 32}
	dotLimit       = limit{min: 1, max: 16}
)

func Shapes() []Shape {
	names := make([]Shape, len(shapes))
	for i, entry := range shapes {
		names[i] = entry.shape
	}

	return names
}

func Draw(s Style) (*image.RGBA, error) {
	s = s.clamped()

	ps, err := partsOf(s)
	if err != nil {
		return nil, err
	}

	reach := ps.reach()
	size := 2*reach + ps.parity
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	ink := paint(s.Color, s.Opacity)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if ps.covers(x-reach, y-reach) {
				img.SetRGBA(x, y, ink)
			}
		}
	}

	return img, nil
}

func (s Style) clamped() Style {
	s.Opacity = opacityLimit.clamp(s.Opacity)
	s.Length = lengthLimit.clamp(s.Length)
	s.Thickness = thicknessLimit.clamp(s.Thickness)
	s.Gap = gapLimit.clamp(s.Gap)
	s.DotSize = dotLimit.clamp(s.DotSize)

	return s
}

func (l limit) clamp(v int) int {
	return min(max(v, l.min), l.max)
}

func partsOf(s Style) (parts, error) {
	for _, entry := range shapes {
		if entry.shape == s.Shape {
			return entry.parts(s), nil
		}
	}

	return parts{}, fmt.Errorf("%w: %q", ErrUnknownShape, s.Shape)
}

// Offsets count from the anchor pixel at half the image size: parity 1 puts the centre in
// the middle of that pixel, parity 0 on its top left corner.
type parts struct {
	parity int
	rects  []rect
}

type rect struct {
	x, y span
}

type span struct {
	lo, hi int
}

func withArms(s Style, up bool) parts {
	parity := s.Thickness % 2
	band := centred(s.Thickness, parity)
	after := span{lo: s.Gap, hi: s.Gap + s.Length - 1}
	before := span{lo: parity - 1 - after.hi, hi: parity - 1 - after.lo}

	rects := []rect{{x: after, y: band}, {x: before, y: band}, {x: band, y: after}}
	if up {
		rects = append(rects, rect{x: band, y: before})
	}

	return parts{parity: parity, rects: rects}
}

func withDot(ps parts, s Style) parts {
	dot := centred(s.DotSize, ps.parity)

	return parts{parity: ps.parity, rects: append(slices.Clone(ps.rects), rect{x: dot, y: dot})}
}

// When n and the parity disagree the spare half pixel goes right and down; the shift floors
// where division would truncate toward zero.
func centred(n, parity int) span {
	lo := (parity - n + 1) >> 1

	return span{lo: lo, hi: lo + n - 1}
}

func (ps parts) reach() int {
	reach := 0
	for _, r := range ps.rects {
		reach = max(reach, -r.x.lo, -r.y.lo, r.x.hi+1-ps.parity, r.y.hi+1-ps.parity)
	}

	return reach
}

func (ps parts) covers(dx, dy int) bool {
	for _, r := range ps.rects {
		if r.x.lo <= dx && dx <= r.x.hi && r.y.lo <= dy && dy <= r.y.hi {
			return true
		}
	}

	return false
}

func paint(c RGB, opacity int) color.RGBA {
	return color.RGBA{R: fade(c.R, opacity), G: fade(c.G, opacity), B: fade(c.B, opacity), A: fade(255, opacity)}
}

func fade(v uint8, percent int) uint8 {
	return uint8((int(v)*percent + 50) / 100)
}
