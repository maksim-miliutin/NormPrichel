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
	Dot       Shape = "dot"
	Cross     Shape = "cross"
	CrossDot  Shape = "cross-dot"
	Circle    Shape = "circle"
	CircleDot Shape = "circle-dot"
	Tee       Shape = "t"
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
	Radius    int
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
	{Circle, func(s Style) parts { return withRing(s) }},
	{CircleDot, func(s Style) parts { return withDot(withRing(s), s) }},
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
	radiusLimit    = limit{min: 2, max: 64}
	dotLimit       = limit{min: 1, max: 16}
)

// Each pixel is sampled on a samples by samples grid, counted in steps of 1/(2*samples) of a
// pixel: every sample lands on an odd step, so a mirrored pixel sees exactly mirrored samples.
const samples = 16

const full = samples * samples

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

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, paint(s.Color, s.Opacity, ps.coverage(x-reach, y-reach)))
		}
	}

	return img, nil
}

func (s Style) clamped() Style {
	s.Opacity = opacityLimit.clamp(s.Opacity)
	s.Length = lengthLimit.clamp(s.Length)
	s.Thickness = thicknessLimit.clamp(s.Thickness)
	s.Gap = gapLimit.clamp(s.Gap)
	s.Radius = radiusLimit.clamp(s.Radius)
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
	rings  []ring
}

type rect struct {
	x, y span
}

type span struct {
	lo, hi int
}

// Radii in half pixels, so an odd thickness stays whole.
type ring struct {
	inner, outer int
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

func withRing(s Style) parts {
	band := ring{inner: max(0, 2*s.Radius-s.Thickness), outer: 2*s.Radius + s.Thickness}

	return parts{parity: s.Thickness % 2, rings: []ring{band}}
}

func withDot(ps parts, s Style) parts {
	dot := centred(s.DotSize, ps.parity)
	rects := append(slices.Clone(ps.rects), rect{x: dot, y: dot})

	return parts{parity: ps.parity, rects: rects, rings: ps.rings}
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
	for _, r := range ps.rings {
		reach = max(reach, (r.outer-ps.parity)/2)
	}

	return reach
}

func (ps parts) coverage(dx, dy int) int {
	if ps.inRect(dx, dy) {
		return full
	}
	if len(ps.rings) == 0 {
		return 0
	}

	count := 0
	for i := 0; i < full; i++ {
		x := 2*samples*dx - samples*ps.parity + 2*(i%samples) + 1
		y := 2*samples*dy - samples*ps.parity + 2*(i/samples) + 1
		if ps.inRing(x*x + y*y) {
			count++
		}
	}

	return count
}

func (ps parts) inRect(dx, dy int) bool {
	for _, r := range ps.rects {
		if r.x.lo <= dx && dx <= r.x.hi && r.y.lo <= dy && dy <= r.y.hi {
			return true
		}
	}

	return false
}

func (ps parts) inRing(squared int) bool {
	for _, r := range ps.rings {
		inner, outer := r.inner*samples, r.outer*samples
		if inner*inner <= squared && squared <= outer*outer {
			return true
		}
	}

	return false
}

func paint(c RGB, opacity, cover int) color.RGBA {
	return color.RGBA{
		R: shade(c.R, opacity, cover),
		G: shade(c.G, opacity, cover),
		B: shade(c.B, opacity, cover),
		A: shade(255, opacity, cover),
	}
}

func shade(v uint8, percent, cover int) uint8 {
	return uint8(roundedRatio(int(v)*percent*cover, 100*full))
}

func roundedRatio(num, den int) int {
	return (2*num + den) / (2 * den)
}
