package crosshair

import (
	"fmt"
	"image"
	"image/color"
	"slices"
)

type Part string

const (
	InnerPart   Part = "inner"
	OuterPart   Part = "outer"
	DotPart     Part = "dot"
	RingPart    Part = "ring"
	OutlinePart Part = "outline"
)

type Parts struct {
	Inner          Lines `json:"inner"`
	Outer          Lines `json:"outer"`
	Dot            Spot  `json:"dot"`
	Ring           Band  `json:"ring"`
	OutlineOpacity int   `json:"outlineOpacity"` // percent
}

type Lines struct {
	On        bool `json:"on"`
	Opacity   int  `json:"opacity"` // percent
	Length    int  `json:"length"`
	Thickness int  `json:"thickness"`
	Gap       int  `json:"gap"`
	Hidden    Arms `json:"hidden,omitzero"`
}

type Arms struct {
	Up    bool `json:"up,omitempty"`
	Right bool `json:"right,omitempty"`
	Down  bool `json:"down,omitempty"`
	Left  bool `json:"left,omitempty"`
}

type Spot struct {
	On      bool `json:"on"`
	Opacity int  `json:"opacity"` // percent
	Size    int  `json:"size"`
}

type Band struct {
	On        bool `json:"on"`
	Opacity   int  `json:"opacity"` // percent
	Radius    int  `json:"radius"`
	Thickness int  `json:"thickness"`
}

var (
	shareLimit = limit{min: 0, max: 100}
	reachLimit = limit{min: 0, max: 64}
)

var partKnobs = []struct {
	part  Part
	name  string
	limit limit
	value func(s *Style) *int
}{
	{InnerPart, "opacity", shareLimit, func(s *Style) *int { return &s.Parts.Inner.Opacity }},
	{InnerPart, "length", lengthLimit, func(s *Style) *int { return &s.Parts.Inner.Length }},
	{InnerPart, "thickness", thicknessLimit, func(s *Style) *int { return &s.Parts.Inner.Thickness }},
	{InnerPart, "gap", reachLimit, func(s *Style) *int { return &s.Parts.Inner.Gap }},
	{OuterPart, "opacity", shareLimit, func(s *Style) *int { return &s.Parts.Outer.Opacity }},
	{OuterPart, "length", lengthLimit, func(s *Style) *int { return &s.Parts.Outer.Length }},
	{OuterPart, "thickness", thicknessLimit, func(s *Style) *int { return &s.Parts.Outer.Thickness }},
	{OuterPart, "gap", reachLimit, func(s *Style) *int { return &s.Parts.Outer.Gap }},
	{DotPart, "opacity", shareLimit, func(s *Style) *int { return &s.Parts.Dot.Opacity }},
	{DotPart, "dot", dotLimit, func(s *Style) *int { return &s.Parts.Dot.Size }},
	{RingPart, "opacity", shareLimit, func(s *Style) *int { return &s.Parts.Ring.Opacity }},
	{RingPart, "radius", radiusLimit, func(s *Style) *int { return &s.Parts.Ring.Radius }},
	{RingPart, "thickness", thicknessLimit, func(s *Style) *int { return &s.Parts.Ring.Thickness }},
	{OutlinePart, "outline", outlineLimit, func(s *Style) *int { return &s.Outline.Width }},
	{OutlinePart, "outlineOpacity", shareLimit, func(s *Style) *int { return &s.Parts.OutlineOpacity }},
}

type layer struct {
	shape   parts
	opacity int
}

// Every part shares the centre of the first one shown, as the dot of a ready shape shares the
// centre of its lines; a part of the other parity gives its spare half pixel right and down.
func (p Parts) parity() int {
	switch {
	case p.Inner.On:
		return p.Inner.Thickness % 2
	case p.Outer.On:
		return p.Outer.Thickness % 2
	case p.Dot.On:
		return p.Dot.Size % 2
	}

	return p.Ring.Thickness % 2
}

func (p Parts) layers() []layer {
	parity := p.parity()
	var out []layer
	if p.Ring.On {
		band := ring{inner: max(0, 2*p.Ring.Radius-p.Ring.Thickness), outer: 2*p.Ring.Radius + p.Ring.Thickness}
		out = append(out, layer{parts{parity: parity, rings: []ring{band}}, p.Ring.Opacity})
	}
	for _, l := range []Lines{p.Outer, p.Inner} {
		if l.On {
			out = append(out, layer{parts{parity: parity, rects: l.arms(parity)}, l.Opacity})
		}
	}
	if p.Dot.On {
		dot := centred(p.Dot.Size, parity)
		out = append(out, layer{parts{parity: parity, rects: []rect{{x: dot, y: dot}}}, p.Dot.Opacity})
	}

	return merged(out)
}

// Parts of one opacity are one layer, so where they cross they do not darken each other.
func merged(layers []layer) []layer {
	var out []layer
	for _, l := range layers {
		i := slices.IndexFunc(out, func(m layer) bool { return m.opacity == l.opacity })
		if i < 0 {
			out = append(out, l)
			continue
		}
		out[i].shape = joined(out[i].shape, l.shape)
	}

	return out
}

func (l Lines) arms(parity int) []rect {
	band := centred(l.Thickness, parity)
	after := span{lo: l.Gap, hi: l.Gap + l.Length - 1}
	before := span{lo: parity - 1 - after.hi, hi: parity - 1 - after.lo}

	var rects []rect
	for _, arm := range []struct {
		hidden bool
		at     rect
	}{
		{l.Hidden.Up, rect{x: band, y: before}},
		{l.Hidden.Right, rect{x: after, y: band}},
		{l.Hidden.Down, rect{x: band, y: after}},
		{l.Hidden.Left, rect{x: before, y: band}},
	} {
		if !arm.hidden {
			rects = append(rects, arm.at)
		}
	}

	return rects
}

// Each part lays its own opacity over the ones before it. The outline fills only what lies round
// the parts, so a clear part shows a hollow edge, the way a game draws it.
func (s Style) assembled() *image.RGBA {
	layers := s.Parts.layers()
	rounds := s.Corners != Sharp || s.Rounding > 0
	parity := s.Parts.parity()
	whole := parts{parity: parity}
	fills := make([]cover, len(layers))
	opacities := make([]int, len(layers))
	for i, l := range layers {
		whole = joined(whole, l.shape)
		fills[i], opacities[i] = l.shape, l.opacity
		if rounds {
			fills[i] = roundedFrom(l.shape, s)
		}
	}
	edge := cover(whole.grown(s.Outline.Width))
	if rounds {
		edge = roundedFrom(whole, s).grown(s.Outline.Width)
	}

	reach := max(1, edge.reach()) + s.blur()
	size := 2*reach + parity
	edges := s.edges(edge, reach, size)
	all := full * hundreds(len(layers))
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			through, covered := shaded(fills, opacities, x-reach, y-reach)
			border := max(0, edges.values[y*size+x]-covered)
			img.SetRGBA(x, y, s.over(100*(all-through), border*s.Parts.OutlineOpacity*(all/full), 100*all))
		}
	}

	return img
}

func hundreds(n int) int {
	out := 1
	for range n {
		out *= 100
	}

	return out
}

// shaded sums the light the layers let through a pixel's samples, 100-opacity of every hundred
// each, and counts the samples covered; only two layers covering part of it need each sample.
func shaded(fills []cover, opacities []int, dx, dy int) (through, covered int) {
	coverages := make([]int, len(fills))
	partly, base := -1, 1
	for i, f := range fills {
		coverages[i] = f.coverage(dx, dy)
		switch {
		case coverages[i] == full:
			base, covered = base*(100-opacities[i]), full
		case coverages[i] == 0:
			base *= 100
		case partly >= 0:
			return sampled(fills, opacities, dx, dy)
		default:
			partly = i
		}
	}
	if partly < 0 {
		return full * base, covered
	}

	c, o := coverages[partly], opacities[partly]

	return base * (c*(100-o) + (full-c)*100), max(covered, c)
}

func sampled(fills []cover, opacities []int, dx, dy int) (through, covered int) {
	for i := range full {
		light, hit := 1, false
		for k, f := range fills {
			if !f.sampled(dx, dy, i) {
				light *= 100
				continue
			}
			light, hit = light*(100-opacities[k]), true
		}
		through += light
		if hit {
			covered++
		}
	}

	return through, covered
}

func (s Style) over(fill, border, whole int) color.RGBA {
	mix := func(f, e uint8) uint8 {
		return uint8(roundedRatio(int(f)*fill+int(e)*border, whole))
	}

	return color.RGBA{R: mix(s.Color.R, s.Outline.Color.R), G: mix(s.Color.G, s.Outline.Color.G), B: mix(s.Color.B, s.Outline.Color.B), A: mix(255, 255)}
}

type Arm string

const (
	Up    Arm = "up"
	Right Arm = "right"
	Down  Arm = "down"
	Left  Arm = "left"
)

func PartNames() []Part {
	return []Part{InnerPart, OuterPart, DotPart, RingPart, OutlinePart}
}

func ArmNames() []Arm {
	return []Arm{Up, Right, Down, Left}
}

func PartKnobs(p Part) []Knob {
	var offered []Knob
	for _, k := range partKnobs {
		if k.part == p {
			offered = append(offered, Knob{Name: k.name, Min: k.limit.min, Max: k.limit.max, Step: 1})
		}
	}

	return offered
}

func (s Style) PartValue(p Part, name string) int {
	for _, k := range partKnobs {
		if k.part == p && k.name == name {
			return *k.value(&s)
		}
	}

	return 0
}

func (s Style) WithPartValue(p Part, name string, v int) (Style, error) {
	for _, k := range partKnobs {
		if k.part == p && k.name == name {
			*k.value(&s) = k.limit.clamp(v)
			return s, nil
		}
	}

	return s, fmt.Errorf("%w: %q of %q", ErrUnknownKnob, name, p)
}

// The outline has no switch of its own: a width of nothing is how it goes.
func (s *Style) switchOf(p Part) *bool {
	switch p {
	case InnerPart:
		return &s.Parts.Inner.On
	case OuterPart:
		return &s.Parts.Outer.On
	case DotPart:
		return &s.Parts.Dot.On
	case RingPart:
		return &s.Parts.Ring.On
	}

	return nil
}

func (s Style) PartShown(p Part) bool {
	on := s.switchOf(p)

	return on != nil && *on
}

func (s Style) WithPartShown(p Part, shown bool) Style {
	if on := s.switchOf(p); on != nil {
		*on = shown
	}

	return s
}

func (s *Style) hiddenOf(p Part, a Arm) *bool {
	lines := map[Part]*Lines{InnerPart: &s.Parts.Inner, OuterPart: &s.Parts.Outer}[p]
	if lines == nil {
		return nil
	}

	return map[Arm]*bool{Up: &lines.Hidden.Up, Right: &lines.Hidden.Right, Down: &lines.Hidden.Down, Left: &lines.Hidden.Left}[a]
}

func (s Style) ArmShown(p Part, a Arm) bool {
	hidden := s.hiddenOf(p, a)

	return hidden != nil && !*hidden
}

func (s Style) WithArm(p Part, a Arm, shown bool) Style {
	if hidden := s.hiddenOf(p, a); hidden != nil {
		*hidden = !shown
	}

	return s
}

func Switched(p Part) bool {
	var s Style

	return s.switchOf(p) != nil
}

func Armed(p Part) bool {
	var s Style

	return s.hiddenOf(p, Up) != nil
}
