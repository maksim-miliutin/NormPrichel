package crosshair

import "slices"

type Corners string

const (
	Sharp Corners = ""
	Soft  Corners = "soft"
	Round Corners = "round"
)

// The shapes drawn of boxes, which rounding turns into boxes with round corners.
var roundable = []Shape{Dot, Cross, CrossDot, Tee, Dots, CircleDot, CircleCross}

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

type figure struct {
	parity  int
	strokes []stroke
	rings   []ring
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

	return figure{parity: f.parity, strokes: strokes, rings: parts{parity: f.parity, rings: f.rings}.grown(width).rings}
}

func (f figure) reach() int {
	reach := parts{parity: f.parity, rings: f.rings}.reach()
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
	if len(near) == 0 && len(f.rings) == 0 {
		return 0
	}

	rings := parts{parity: f.parity, rings: f.rings}
	count := 0
	for i := 0; i < full; i++ {
		x, y := x0+2*(i%samples)+1, y0+2*(i/samples)+1
		if slices.ContainsFunc(near, func(k stroke) bool { return k.holds(x, y) }) || len(f.rings) > 0 && rings.inRing(x, y) {
			count++
		}
	}

	return count
}
