package crosshair

import (
	"errors"
	"fmt"
	"slices"
)

var ErrUnknownKnob = errors.New("crosshair: there is no such value")

type Knob struct {
	Name string
	Min  int
	Max  int
	Step int
}

var knobs = []struct {
	name  string
	limit limit
	value func(s *Style) *int
}{
	{"opacity", opacityLimit, func(s *Style) *int { return &s.Opacity }},
	{"length", lengthLimit, func(s *Style) *int { return &s.Length }},
	{"thickness", thicknessLimit, func(s *Style) *int { return &s.Thickness }},
	{"gap", gapLimit, func(s *Style) *int { return &s.Gap }},
	{"radius", radiusLimit, func(s *Style) *int { return &s.Radius }},
	{"dot", dotLimit, func(s *Style) *int { return &s.DotSize }},
	{"outline", outlineLimit, func(s *Style) *int { return &s.Outline.Width }},
}

func Knobs(shape Shape) []Knob {
	for _, entry := range shapes {
		if entry.shape == shape {
			return knobsOf(entry.uses)
		}
	}

	return nil
}

func knobsOf(used []string) []Knob {
	var offered []Knob
	for _, k := range knobs {
		if slices.Contains(used, k.name) {
			offered = append(offered, Knob{Name: k.name, Min: k.limit.min, Max: k.limit.max, Step: 1})
		}
	}

	return offered
}

func PictureScale() Knob {
	return Knob{Name: "scale", Min: scaleLimit.min, Max: scaleLimit.max, Step: 25}
}

func (s Style) Value(name string) int {
	for _, k := range knobs {
		if k.name == name {
			return *k.value(&s)
		}
	}

	return 0
}

func (s Style) WithValue(name string, v int) (Style, error) {
	for _, k := range knobs {
		if k.name == name {
			*k.value(&s) = k.limit.clamp(v)
			return s, nil
		}
	}

	return s, fmt.Errorf("%w: %q", ErrUnknownKnob, name)
}
