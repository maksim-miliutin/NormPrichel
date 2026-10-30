package crosshair

import (
	"math"
	"slices"
)

type Corners string

const (
	Sharp Corners = ""
	Soft  Corners = "soft"
	Round Corners = "round"
)

type rounder struct {
	shape  Shape
	figure func(s Style, sharp parts) figure
}

var rounders = []rounder{
	{Dot, boxesOf}, {Cross, boxesOf}, {CrossDot, boxesOf}, {Tee, boxesOf}, {Dots, boxesOf}, {CircleDot, boxesOf}, {CircleCross, boxesOf},
	{Square, framed{}.figure},
	{SquareDot, framed{dotted: true}.figure},
	{Brackets, framed{openAcross: true, dotted: true}.figure},
	{Viewfinder, framed{openAcross: true, openDown: true, dotted: true}.figure},
	{X, diagonals{directions: corners, fromGap: true}.figure},
	{XDot, diagonals{directions: corners, fromGap: true, dotted: true}.figure},
	{Chevron, diagonals{directions: lowerCorners}.figure},
	{Horseshoe, horseshoeOf},
	{Diamond, func(s Style, _ parts) figure {
		return polygonOf(s, []point{{0, -s.Radius}, {s.Radius, 0}, {0, s.Radius}, {-s.Radius, 0}})
	}},
	{Triangle, func(s Style, _ parts) figure {
		base := s.Radius / 2
		half := (s.Radius + base) / 2
		return polygonOf(s, []point{{0, -s.Radius}, {half, base}, {-half, base}})
	}},
}

// Soft corners round a line of their thickness as far as this share would, and stop short of
// either end, being neither sharp nor round.
func (s Style) Roundness() int {
	c := s.clamped()
	switch {
	case c.Rounding > 0:
		return c.Rounding
	case c.Corners == Round:
		return roundingLimit.max
	case c.Corners == Soft:
		return limit{min: 1, max: roundingLimit.max - 1}.clamp(roundedRatio(2*roundingLimit.max, c.Thickness))
	}

	return 0
}

func (s Style) Rounded(share int) Style {
	s.Corners, s.Rounding = Sharp, roundingLimit.clamp(share)

	return s.held()
}

func Rounds(shape Shape) bool {
	return slices.ContainsFunc(rounders, func(r rounder) bool { return r.shape == shape })
}

func roundedFigure(s Style, sharp parts) (figure, bool) {
	for _, entry := range rounders {
		if entry.shape == s.Shape {
			f := entry.figure(s, sharp)
			f.least = sharp.reach()
			return f, true
		}
	}

	return figure{}, false
}

// A rounded drawing is measured in the steps its samples fall on and tested in whole numbers
// only, so every machine draws it to the same bit.
const unit = 2 * samples

type point struct {
	x, y int
}

// A stroke is a box in the frame of a whole-number direction (dx, dy), along u = dx*x + dy*y and
// across v = dx*y - dy*x, both stretched by the length of (dx, dy); it holds all within r of its core.
type stroke struct {
	dx, dy         int
	u0, u1, v0, v1 int
	r              int
	low, high      point
}

func boxed(low, high point, r int) stroke {
	return stroke{dx: 1, u0: low.x + r, u1: high.x - r, v0: low.y + r, v1: high.y - r, r: r, low: low, high: high}
}

// A segment runs the whole-number direction dir from one end to the other, both in units, half to
// either side and past either end; off the axes its stretched sizes round to whole units.
type segment struct {
	dir, from, to point
	half, corner  int
}

func (sg segment) stroke() stroke {
	d, r := sg.dir, sg.corner
	n2 := d.x*d.x + d.y*d.y
	core := stretch(sg.half-r, n2)
	across := d.x*sg.from.y - d.y*sg.from.x
	k := stroke{dx: d.x, dy: d.y, r: r, v0: across - core, v1: across + core,
		u0: d.x*sg.from.x + d.y*sg.from.y - core, u1: d.x*sg.to.x + d.y*sg.to.y + core}

	var xs, ys []int
	for _, u := range []int{k.u0, k.u1} {
		for _, v := range []int{k.v0, k.v1} {
			xs, ys = append(xs, d.x*u-d.y*v), append(ys, d.y*u+d.x*v)
		}
	}
	k.low = point{floorDiv(slices.Min(xs), n2) - r, floorDiv(slices.Min(ys), n2) - r}
	k.high = point{-floorDiv(-slices.Max(xs), n2) + r, -floorDiv(-slices.Max(ys), n2) + r}

	return k
}

// stretch rounds k times the root of n2 to a whole number by whole-number steps only.
func stretch(k, n2 int) int {
	m := k * k * n2
	root := int(math.Sqrt(float64(m)))
	for root*root > m {
		root--
	}
	for (root+1)*(root+1) <= m {
		root++
	}
	if 4*m >= (2*root+1)*(2*root+1) {
		root++
	}

	return root
}

func boxesOf(s Style, sharp parts) figure {
	return roundedFrom(sharp, s)
}

// Round corners stop short of the openings and caps close their ends: openings keep the sharp width.
type framed struct {
	openAcross, openDown, dotted bool
}

func (fm framed) figure(s Style, sharp parts) figure {
	if s.Corners == Soft {
		return boxesOf(s, sharp)
	}

	parity := s.Thickness % 2
	side, h := s.Radius*unit, s.Thickness*unit/2
	cut := min(unit*s.Gap-samples*parity+h, side-h)
	rr := side / 2
	if fm.openAcross || fm.openDown {
		rr = max(0, min(rr, side-cut))
	}
	fr := frame{side: side, h: h, outer: bend(s, rr+h)}
	f := figure{parity: parity}
	for _, sign := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		if fm.openAcross {
			fr.cutX = cut
			f.strokes = append(f.strokes, capped(point{sign[0] * cut, sign[1] * side}, h, bend(s, h)))
		}
		if fm.openDown {
			fr.cutY = cut
			f.strokes = append(f.strokes, capped(point{sign[0] * side, sign[1] * cut}, h, bend(s, h)))
		}
	}
	f.frames = []frame{fr}
	if fm.dotted {
		f.strokes = append(f.strokes, roundedFrom(withDot(parts{parity: parity}, s), s).strokes...)
	}

	return f
}

func capped(at point, h, r int) stroke {
	return boxed(point{at.x - h, at.y - h}, point{at.x + h, at.y + h}, r)
}

// Corners come in pixels. A middle left narrower than the outline and a pixel would show only
// flecks of outline, so a small thick figure is filled, as its sharp drawing is.
func polygonOf(s Style, corners []point) figure {
	parity := s.Thickness % 2
	h := s.Thickness * unit / 2
	f := figure{parity: parity}
	var inside hull
	twice, around := 0, 0.0
	for i, from := range corners {
		to := corners[(i+1)%len(corners)]
		dir := point{to.x - from.x, to.y - from.y}
		start, end := point{unit * from.x, unit * from.y}, point{unit * to.x, unit * to.y}
		f.strokes = append(f.strokes, segment{dir: dir, from: start, to: end, half: h, corner: bend(s, h)}.stroke())
		inside = append(inside, start)
		twice += from.x*to.y - to.x*from.y
		around += math.Hypot(float64(to.x-from.x), float64(to.y-from.y))
	}
	if float64(twice)/around-float64(s.Thickness)/2 < float64(s.Outline.Width+1) {
		f.hulls = []hull{inside}
	}
	f.strokes = append(f.strokes, roundedFrom(withDot(parts{parity: parity}, s), s).strokes...)

	return f
}

// The cut ring ends on the lower diagonals, a radius out from the middle. A cap is the band
// across the end swept r either way of the cut, which keeps its back half within the ring.
func horseshoeOf(s Style, sharp parts) figure {
	f := boxesOf(s, sharp)
	if s.Corners == Soft {
		return f
	}

	h, aside := s.Thickness*unit/2, stretch(samples*s.Radius, 2)
	r := bend(s, h)
	off := stretch(h-r, 2) / 2
	for _, side := range []int{1, -1} {
		from, to := point{side * (aside - off), aside - off}, point{side * (aside + off), aside + off}
		f.strokes = append(f.strokes, segment{dir: point{side, 1}, from: from, to: to, half: r, corner: r}.stroke())
	}

	return f
}

// The smooth arms run where the sharp ones step their brush, as wide and half of it longer at each end.
type diagonals struct {
	directions      [][2]int
	fromGap, dotted bool
}

func (dg diagonals) figure(s Style, _ parts) figure {
	from := 0
	if dg.fromGap {
		from = s.Gap
	}
	parity := s.Thickness % 2
	h := s.Thickness * unit / 2
	f := figure{parity: parity}
	for _, d := range dg.directions {
		dir := point{d[0], d[1]}
		first := point{unit * d[0] * from, unit * d[1] * from}
		last := point{unit * d[0] * (from + s.Length - 1), unit * d[1] * (from + s.Length - 1)}
		f.strokes = append(f.strokes, segment{dir: dir, from: first, to: last, half: h, corner: bend(s, h)}.stroke())
	}
	if dg.dotted {
		dot := roundedFrom(withDot(parts{parity: parity}, s), s)
		f.strokes = append(f.strokes, dot.strokes...)
	}

	return f
}

func (k stroke) holds(x, y int) bool {
	u, v := k.dx*x+k.dy*y, k.dx*y-k.dy*x
	du, dv := max(k.u0-u, 0, u-k.u1), max(k.v0-v, 0, v-k.v1)

	return du*du+dv*dv <= (k.dx*k.dx+k.dy*k.dy)*k.r*k.r
}

func (k stroke) grown(by int) stroke {
	k.r += by
	k.low, k.high = point{k.low.x - by, k.low.y - by}, point{k.high.x + by, k.high.y + by}

	return k
}

// A frame is the band between a square of half side side+h whose corners curve with radius outer
// and one of half side side-h curving 2h less, less what lies within cutX across or cutY down.
type frame struct {
	side, h, outer, cutX, cutY int
}

func (f frame) holds(x, y int) bool {
	ax, ay := max(x, -x), max(y, -y)
	if ax < f.cutX || ay < f.cutY {
		return false
	}

	return within(ax, ay, f.side+f.h, f.outer) && !inside(ax, ay, f.side-f.h, max(f.outer-2*f.h, 0))
}

func within(ax, ay, half, r int) bool {
	qx, qy := max(ax-half+r, 0), max(ay-half+r, 0)

	return ax <= half && ay <= half && qx*qx+qy*qy <= r*r
}

// inside leaves out the edge itself, which belongs to the band as the outer edge does.
func inside(ax, ay, half, r int) bool {
	qx, qy := max(ax-half+r, 0), max(ay-half+r, 0)

	return ax < half && ay < half && (r == 0 || qx*qx+qy*qy < r*r)
}

// The least reach keeps a rounded drawing the size of the sharp one, which may reach past what
// it draws.
type figure struct {
	parity, least int
	strokes       []stroke
	frames        []frame
	hulls         []hull
	rings         []ring
}

// A hull fills the convex polygon its corners go round clockwise, in units.
type hull []point

func (hl hull) holds(x, y int) bool {
	for i, a := range hl {
		b := hl[(i+1)%len(hl)]
		if (b.x-a.x)*(y-a.y)-(b.y-a.y)*(x-a.x) < 0 {
			return false
		}
	}

	return true
}

func roundedFrom(ps parts, s Style) figure {
	var strokes []stroke
	for _, r := range ps.rects {
		low := point{unit*r.x.lo - samples*ps.parity, unit*r.y.lo - samples*ps.parity}
		high := point{unit*(r.x.hi+1) - samples*ps.parity, unit*(r.y.hi+1) - samples*ps.parity}
		if high.x <= low.x || high.y <= low.y {
			continue
		}
		strokes = append(strokes, boxed(low, high, bend(s, min(high.x-low.x, high.y-low.y)/2)))
	}

	return figure{parity: ps.parity, strokes: strokes, rings: ps.rings}
}

// bend is the radius a corner takes of the most it can have: soft corners round off a pixel,
// round ones all of it, and a share of the way to round that share of it.
func bend(s Style, most int) int {
	switch {
	case s.Rounding > 0:
		return most * s.Rounding / roundingLimit.max
	case s.Corners == Soft:
		return min(unit, most)
	case s.Corners == Round:
		return most
	}

	return 0
}

func (f figure) grown(width int) figure {
	strokes := make([]stroke, len(f.strokes))
	for i, k := range f.strokes {
		strokes[i] = k.grown(width * unit)
	}
	frames := make([]frame, len(f.frames))
	for i, fr := range f.frames {
		fr.h += width * unit
		fr.outer += width * unit
		frames[i] = fr
	}

	rings := parts{parity: f.parity, rings: f.rings}.grown(width).rings

	return figure{parity: f.parity, least: f.least + width, strokes: strokes, frames: frames, hulls: f.hulls, rings: rings}
}

func (f figure) reach() int {
	reach := max(f.least, parts{parity: f.parity, rings: f.rings}.reach())
	for _, k := range f.strokes {
		low := floorDiv(min(k.low.x, k.low.y)+samples*f.parity, unit)
		high := -floorDiv(-(max(k.high.x, k.high.y) + samples*f.parity), unit)
		reach = max(reach, -low, high-f.parity)
	}

	return reach
}

func floorDiv(a, b int) int {
	if a < 0 {
		return -((-a + b - 1) / b)
	}

	return a / b
}

func (f figure) coverage(dx, dy int) int {
	x0, y0 := unit*dx-samples*f.parity, unit*dy-samples*f.parity
	var near []stroke
	for _, k := range f.strokes {
		if k.low.x < x0+unit && k.high.x > x0 && k.low.y < y0+unit && k.high.y > y0 {
			near = append(near, k)
		}
	}
	if len(near) == 0 && len(f.frames) == 0 && len(f.hulls) == 0 && len(f.rings) == 0 {
		return 0
	}

	rings := parts{parity: f.parity, rings: f.rings}
	count := 0
	for i := 0; i < full; i++ {
		x, y := x0+2*(i%samples)+1, y0+2*(i/samples)+1
		if slices.ContainsFunc(near, func(k stroke) bool { return k.holds(x, y) }) ||
			slices.ContainsFunc(f.frames, func(fr frame) bool { return fr.holds(x, y) }) ||
			slices.ContainsFunc(f.hulls, func(hl hull) bool { return hl.holds(x, y) }) ||
			len(f.rings) > 0 && rings.inRing(x, y) {
			count++
		}
	}

	return count
}
