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

var rounders = []struct {
	shape  Shape
	figure func(s Style, sharp parts) figure
}{
	{Dot, boxesOf}, {Cross, boxesOf}, {CrossDot, boxesOf}, {Tee, boxesOf}, {Dots, boxesOf}, {CircleDot, boxesOf}, {CircleCross, boxesOf},
	{Square, framed{}.figure},
	{SquareDot, framed{dotted: true}.figure},
	{Brackets, framed{openAcross: true, dotted: true}.figure},
	{Viewfinder, framed{openAcross: true, openDown: true, dotted: true}.figure},
	{X, diagonals{directions: corners, fromGap: true}.figure},
	{XDot, diagonals{directions: corners, fromGap: true, dotted: true}.figure},
	{Chevron, diagonals{directions: lowerCorners}.figure},
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
	return roundedFrom(sharp, s.Corners)
}

// Round corners stop short of the openings and discs cap their ends: openings keep the sharp width.
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
	fr := frame{q: side - rr, rr: rr, h: h}
	f := figure{parity: parity}
	for _, sign := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		if fm.openAcross {
			fr.cutX = cut
			f.strokes = append(f.strokes, disc(point{sign[0] * cut, sign[1] * side}, h))
		}
		if fm.openDown {
			fr.cutY = cut
			f.strokes = append(f.strokes, disc(point{sign[0] * side, sign[1] * cut}, h))
		}
	}
	f.frames = []frame{fr}
	if fm.dotted {
		f.strokes = append(f.strokes, roundedFrom(withDot(parts{parity: parity}, s), s.Corners).strokes...)
	}

	return f
}

func disc(at point, h int) stroke {
	return boxed(point{at.x - h, at.y - h}, point{at.x + h, at.y + h}, h)
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
		f.strokes = append(f.strokes, segment{dir: dir, from: first, to: last, half: h, corner: rounding(s.Corners, h)}.stroke())
	}
	if dg.dotted {
		dot := roundedFrom(withDot(parts{parity: parity}, s), s.Corners)
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

// A frame strokes the line round a square whose corners curve with radius rr from a core of half
// side q: h to either side of it, less what lies within cutX of the middle across or cutY down.
type frame struct {
	q, rr, h, cutX, cutY int
}

func (f frame) holds(x, y int) bool {
	ax, ay := max(x, -x), max(y, -y)
	if ax < f.cutX || ay < f.cutY {
		return false
	}
	qx, qy := max(ax-f.q, 0), max(ay-f.q, 0)
	if qx == 0 && qy == 0 {
		return f.rr+min(f.q-ax, f.q-ay) <= f.h
	}
	far, near := f.rr+f.h, max(f.rr-f.h, 0)

	return near*near <= qx*qx+qy*qy && qx*qx+qy*qy <= far*far
}

// The least reach keeps a rounded drawing the size of the sharp one, which may reach past what
// it draws.
type figure struct {
	parity, least int
	strokes       []stroke
	frames        []frame
	rings         []ring
}

func roundedFrom(ps parts, corners Corners) figure {
	var strokes []stroke
	for _, r := range ps.rects {
		low := point{unit*r.x.lo - samples*ps.parity, unit*r.y.lo - samples*ps.parity}
		high := point{unit*(r.x.hi+1) - samples*ps.parity, unit*(r.y.hi+1) - samples*ps.parity}
		if high.x <= low.x || high.y <= low.y {
			continue
		}
		strokes = append(strokes, boxed(low, high, rounding(corners, min(high.x-low.x, high.y-low.y)/2)))
	}

	return figure{parity: ps.parity, strokes: strokes, rings: ps.rings}
}

// Soft corners round off a pixel, round ones all of half the stroke.
func rounding(corners Corners, half int) int {
	if corners == Soft {
		return min(unit, half)
	}

	return half
}

func (f figure) grown(width int) figure {
	strokes := make([]stroke, len(f.strokes))
	for i, k := range f.strokes {
		strokes[i] = k.grown(width * unit)
	}
	frames := make([]frame, len(f.frames))
	for i, fr := range f.frames {
		fr.h += width * unit
		frames[i] = fr
	}

	return figure{parity: f.parity, least: f.least + width, strokes: strokes, frames: frames, rings: parts{parity: f.parity, rings: f.rings}.grown(width).rings}
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
	if len(near) == 0 && len(f.frames) == 0 && len(f.rings) == 0 {
		return 0
	}

	rings := parts{parity: f.parity, rings: f.rings}
	count := 0
	for i := 0; i < full; i++ {
		x, y := x0+2*(i%samples)+1, y0+2*(i/samples)+1
		if slices.ContainsFunc(near, func(k stroke) bool { return k.holds(x, y) }) ||
			slices.ContainsFunc(f.frames, func(fr frame) bool { return fr.holds(x, y) }) ||
			len(f.rings) > 0 && rings.inRing(x, y) {
			count++
		}
	}

	return count
}
