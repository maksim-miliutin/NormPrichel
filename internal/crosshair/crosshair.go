package crosshair

import (
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"slices"
)

type Shape string

const (
	Dot         Shape = "dot"
	Cross       Shape = "cross"
	CrossDot    Shape = "cross-dot"
	Circle      Shape = "circle"
	CircleDot   Shape = "circle-dot"
	Tee         Shape = "t"
	X           Shape = "x"
	XDot        Shape = "x-dot"
	Square      Shape = "square"
	SquareDot   Shape = "square-dot"
	Chevron     Shape = "chevron"
	CircleCross Shape = "circle-cross"
	Brackets    Shape = "brackets"
	Viewfinder  Shape = "viewfinder"
	Diamond     Shape = "diamond"
)

type RGB struct {
	R, G, B uint8
}

type Style struct {
	Shape     Shape   `json:"shape"`
	Color     RGB     `json:"color"`
	Opacity   int     `json:"opacity"` // percent
	Length    int     `json:"length"`
	Thickness int     `json:"thickness"`
	Gap       int     `json:"gap"`
	Radius    int     `json:"radius"`
	DotSize   int     `json:"dot"`
	Outline   Outline `json:"outline"`
}

type Outline struct {
	Width int `json:"width"` // zero turns the outline off
	Color RGB `json:"color"`
}

var (
	ErrUnknownShape = errors.New("crosshair: unknown shape")
	ErrBadColour    = errors.New("crosshair: a colour is written as #RRGGBB")
)

var shapes = []struct {
	shape Shape
	parts func(s Style) parts
	uses  []string
}{
	{Dot, func(s Style) parts { return withDot(parts{parity: s.DotSize % 2}, s) }, []string{"opacity", "dot", "outline"}},
	{Cross, func(s Style) parts { return withArms(s, true) }, []string{"opacity", "length", "thickness", "gap", "outline"}},
	{CrossDot, func(s Style) parts { return withDot(withArms(s, true), s) }, []string{"opacity", "length", "thickness", "gap", "dot", "outline"}},
	{Circle, func(s Style) parts { return withRing(s) }, []string{"opacity", "thickness", "radius", "outline"}},
	{CircleDot, func(s Style) parts { return withDot(withRing(s), s) }, []string{"opacity", "thickness", "radius", "dot", "outline"}},
	{Tee, func(s Style) parts { return withArms(s, false) }, []string{"opacity", "length", "thickness", "gap", "outline"}},
	{X, func(s Style) parts { return withDiagonals(s, s.Gap, corners) }, []string{"opacity", "length", "thickness", "gap", "outline"}},
	{XDot, func(s Style) parts { return withDot(withDiagonals(s, s.Gap, corners), s) }, []string{"opacity", "length", "thickness", "gap", "dot", "outline"}},
	{Square, func(s Style) parts { return withSquare(s) }, []string{"opacity", "thickness", "radius", "outline"}},
	{SquareDot, func(s Style) parts { return withDot(withSquare(s), s) }, []string{"opacity", "thickness", "radius", "dot", "outline"}},
	{Chevron, func(s Style) parts { return withDiagonals(s, 0, lowerCorners) }, []string{"opacity", "length", "thickness", "outline"}},
	{CircleCross, func(s Style) parts { return joined(withRing(s), withArms(s, true)) }, []string{"opacity", "length", "thickness", "gap", "radius", "outline"}},
	{Brackets, func(s Style) parts { return withDot(withBrackets(s), s) }, []string{"opacity", "thickness", "gap", "radius", "dot", "outline"}},
	{Viewfinder, func(s Style) parts { return withDot(withViewfinder(s), s) }, []string{"opacity", "thickness", "gap", "radius", "dot", "outline"}},
	{Diamond, func(s Style) parts { return withDot(withDiamond(s), s) }, []string{"opacity", "thickness", "radius", "dot", "outline"}},
}

// Directions of diagonal arms, x right and y down: all four corners for the diagonal cross, the
// two lower ones for the chevron, whose tip then points up at the aim.
var (
	corners      = [][2]int{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}}
	lowerCorners = [][2]int{{1, 1}, {-1, 1}}
)

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
	outlineLimit   = limit{min: 0, max: 4}
)

// Each pixel is sampled on a samples by samples grid, counted in steps of 1/(2*samples) of a
// pixel: every sample lands on an odd step, so a mirrored pixel sees exactly mirrored samples.
const samples = 16

const full = samples * samples

func Default() Style {
	return Style{
		Shape:     CrossDot,
		Color:     RGB{R: 0x00, G: 0xFF, B: 0x66},
		Opacity:   100,
		Length:    8,
		Thickness: 2,
		Gap:       3,
		Radius:    10,
		DotSize:   2,
		Outline:   Outline{Width: 1, Color: RGB{}},
	}
}

func (c RGB) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "#%02X%02X%02X", c.R, c.G, c.B), nil
}

func (c *RGB) UnmarshalText(text []byte) error {
	if len(text) != 7 || text[0] != '#' {
		return fmt.Errorf("%w: %q", ErrBadColour, text)
	}

	channels, err := hex.DecodeString(string(text[1:]))
	if err != nil {
		return fmt.Errorf("%w: %q: %w", ErrBadColour, text, err)
	}

	*c = RGB{R: channels[0], G: channels[1], B: channels[2]}

	return nil
}

func Shapes() []Shape {
	names := make([]Shape, len(shapes))
	for i, entry := range shapes {
		names[i] = entry.shape
	}

	return names
}

func Draw(s Style) (*image.RGBA, error) {
	s = s.clamped()

	fill, err := partsOf(s)
	if err != nil {
		return nil, err
	}

	edge := parts{parity: fill.parity}
	if s.Outline.Width > 0 {
		edge = fill.grown(s.Outline.Width)
	}

	reach := max(fill.reach(), edge.reach())
	size := 2*reach + fill.parity
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := x-reach, y-reach
			img.SetRGBA(x, y, s.paint(layered(fill.coverage(dx, dy), edge.coverage(dx, dy))))
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
	s.Outline.Width = outlineLimit.clamp(s.Outline.Width)

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

// A diagonal arm steps a square brush one pixel across and one down at a time, so it stays as
// crisp as a straight arm and its outline is the brushes' outlines.
func withDiagonals(s Style, from int, directions [][2]int) parts {
	parity := s.Thickness % 2
	brush := centred(s.Thickness, parity)

	var rects []rect
	for _, d := range directions {
		for step := from; step < from+s.Length; step++ {
			rects = append(rects, rect{x: brush.shifted(d[0] * step), y: brush.shifted(d[1] * step)})
		}
	}

	return parts{parity: parity, rects: rects}
}

// The radius runs from the centre to the middle of the square's sides, as it does to the middle
// of a ring.
func withSquare(s Style) parts {
	parity := s.Thickness % 2
	band := centred(s.Thickness, parity)
	side := span{lo: band.lo - s.Radius, hi: band.hi + s.Radius}

	return parts{parity: parity, rects: []rect{
		{x: side, y: band.shifted(-s.Radius)},
		{x: side, y: band.shifted(s.Radius)},
		{x: band.shifted(-s.Radius), y: side},
		{x: band.shifted(s.Radius), y: side},
	}}
}

// Brackets are the square with an opening in its top and bottom side; the gap counts from the
// centre the way it does for the arms of a cross.
func withBrackets(s Style) parts {
	parity := s.Thickness % 2
	band := centred(s.Thickness, parity)
	side := span{lo: band.lo - s.Radius, hi: band.hi + s.Radius}
	right := span{lo: s.Gap, hi: side.hi}
	left := span{lo: parity - 1 - right.hi, hi: parity - 1 - right.lo}
	top, bottom := band.shifted(-s.Radius), band.shifted(s.Radius)

	return parts{parity: parity, rects: []rect{
		{x: band.shifted(-s.Radius), y: side},
		{x: band.shifted(s.Radius), y: side},
		{x: left, y: top}, {x: right, y: top},
		{x: left, y: bottom}, {x: right, y: bottom},
	}}
}

func withViewfinder(s Style) parts {
	parity := s.Thickness % 2
	band := centred(s.Thickness, parity)
	after := span{lo: s.Gap, hi: band.hi + s.Radius}
	before := span{lo: parity - 1 - after.hi, hi: parity - 1 - after.lo}

	var rects []rect
	for _, edge := range []span{band.shifted(-s.Radius), band.shifted(s.Radius)} {
		for _, piece := range []span{before, after} {
			rects = append(rects, rect{x: piece, y: edge}, rect{x: edge, y: piece})
		}
	}

	return parts{parity: parity, rects: rects}
}

// The sides run corner to corner the way the arms of the diagonal cross do; the radius reaches a
// corner, where for the square it reaches the middle of a side.
func withDiamond(s Style) parts {
	parity := s.Thickness % 2
	brush := centred(s.Thickness, parity)

	var rects []rect
	for k := range s.Radius {
		for _, at := range [][2]int{{k, k - s.Radius}, {s.Radius - k, k}, {-k, s.Radius - k}, {k - s.Radius, -k}} {
			rects = append(rects, rect{x: brush.shifted(at[0]), y: brush.shifted(at[1])})
		}
	}

	return parts{parity: parity, rects: rects}
}

func joined(a, b parts) parts {
	return parts{parity: a.parity, rects: slices.Concat(a.rects, b.rects), rings: slices.Concat(a.rings, b.rings)}
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

func (ps parts) grown(width int) parts {
	rects := make([]rect, len(ps.rects))
	for i, r := range ps.rects {
		rects[i] = rect{x: r.x.widened(width), y: r.y.widened(width)}
	}

	rings := make([]ring, len(ps.rings))
	for i, r := range ps.rings {
		rings[i] = ring{inner: max(0, r.inner-2*width), outer: r.outer + 2*width}
	}

	return parts{parity: ps.parity, rects: rects, rings: rings}
}

func (s span) shifted(by int) span {
	return span{lo: s.lo + by, hi: s.hi + by}
}

func (s span) widened(width int) span {
	return span{lo: s.lo - width, hi: s.hi + width}
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

// The fill lies over the outline; both weights count in full*full parts of a pixel.
type weights struct {
	fill, edge int
}

func layered(fill, edge int) weights {
	return weights{fill: fill * full, edge: edge * (full - fill)}
}

func (s Style) paint(w weights) color.RGBA {
	return color.RGBA{
		R: s.mix(s.Color.R, s.Outline.Color.R, w),
		G: s.mix(s.Color.G, s.Outline.Color.G, w),
		B: s.mix(s.Color.B, s.Outline.Color.B, w),
		A: s.mix(255, 255, w),
	}
}

func (s Style) mix(fill, edge uint8, w weights) uint8 {
	return uint8(roundedRatio((int(fill)*w.fill+int(edge)*w.edge)*s.Opacity, 100*full*full))
}

func roundedRatio(num, den int) int {
	return (2*num + den) / (2 * den)
}
