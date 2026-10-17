package crosshair_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

func names(knobs []crosshair.Knob) []string {
	var got []string
	for _, k := range knobs {
		got = append(got, k.Name)
	}

	return got
}

func TestEveryShapeOffersTheValuesItIsDrawnFrom(t *testing.T) {
	want := map[crosshair.Shape][]string{
		crosshair.Dot:         {"opacity", "dot", "outline"},
		crosshair.Cross:       {"opacity", "length", "thickness", "gap", "outline"},
		crosshair.CrossDot:    {"opacity", "length", "thickness", "gap", "dot", "outline"},
		crosshair.Circle:      {"opacity", "thickness", "radius", "outline"},
		crosshair.CircleDot:   {"opacity", "thickness", "radius", "dot", "outline"},
		crosshair.Tee:         {"opacity", "length", "thickness", "gap", "outline"},
		crosshair.X:           {"opacity", "length", "thickness", "gap", "outline"},
		crosshair.XDot:        {"opacity", "length", "thickness", "gap", "dot", "outline"},
		crosshair.Square:      {"opacity", "thickness", "radius", "outline"},
		crosshair.SquareDot:   {"opacity", "thickness", "radius", "dot", "outline"},
		crosshair.Chevron:     {"opacity", "length", "thickness", "outline"},
		crosshair.CircleCross: {"opacity", "length", "thickness", "gap", "radius", "outline"},
		crosshair.Brackets:    {"opacity", "thickness", "gap", "radius", "dot", "outline"},
		crosshair.Viewfinder:  {"opacity", "thickness", "gap", "radius", "dot", "outline"},
		crosshair.Diamond:     {"opacity", "thickness", "radius", "dot", "outline"},
		crosshair.Triangle:    {"opacity", "thickness", "radius", "dot", "outline"},
	}

	for _, shape := range crosshair.Shapes() {
		if got := names(crosshair.Knobs(shape)); !slices.Equal(got, want[shape]) {
			t.Errorf("%s offers %v, want %v", shape, got, want[shape])
		}
	}
	if got := crosshair.Knobs("star"); got != nil {
		t.Errorf("an unknown shape offers %v, want nothing", names(got))
	}
}

func TestEveryValueAShapeOffersChangesItsDrawingAndNoOtherDoes(t *testing.T) {
	for _, shape := range crosshair.Shapes() {
		base := crosshair.Default()
		base.Shape = shape
		before, err := crosshair.Draw(base)
		if err != nil {
			t.Fatal(err)
		}

		offered := names(crosshair.Knobs(shape))
		for _, name := range []string{"opacity", "length", "thickness", "gap", "radius", "dot", "outline"} {
			changed, _ := base.WithValue(name, base.Value(name)-1)
			after, _ := crosshair.Draw(changed)
			if differs := !same(before, after); differs != slices.Contains(offered, name) {
				t.Errorf("%s: one step of %s changes the drawing %v, offered %v", shape, name, differs, offered)
			}
		}
	}
}

func TestAValueIsReadAndWrittenWithinItsRange(t *testing.T) {
	for _, knob := range crosshair.Knobs(crosshair.CrossDot) {
		for _, c := range []struct{ asked, want int }{
			{asked: knob.Max + 5, want: knob.Max},
			{asked: knob.Min - 5, want: knob.Min},
			{asked: (knob.Min + knob.Max) / 2, want: (knob.Min + knob.Max) / 2},
		} {
			style, err := crosshair.Default().WithValue(knob.Name, c.asked)
			if err != nil {
				t.Fatal(err)
			}
			if got := style.Value(knob.Name); got != c.want {
				t.Errorf("%s set to %d reads %d, want %d", knob.Name, c.asked, got, c.want)
			}
		}
	}
}

func TestTheOutlineValueIsItsWidthAndTheDotItsSize(t *testing.T) {
	style, _ := crosshair.Default().WithValue("outline", 3)
	style, _ = style.WithValue("dot", 5)

	if style.Outline.Width != 3 || style.DotSize != 5 {
		t.Errorf("got outline %d and dot %d, want 3 and 5", style.Outline.Width, style.DotSize)
	}
}

func TestTheRangesAreTheOnesTheDrawingKeepsTo(t *testing.T) {
	for _, knob := range crosshair.Knobs(crosshair.CrossDot) {
		at, _ := crosshair.Default().WithValue(knob.Name, knob.Max)
		past := at
		switch knob.Name {
		case "opacity":
			past.Opacity++
		case "length":
			past.Length++
		case "thickness":
			past.Thickness++
		case "gap":
			past.Gap++
		case "dot":
			past.DotSize++
		case "outline":
			past.Outline.Width++
		}

		drawnAt, _ := crosshair.Draw(at)
		drawnPast, _ := crosshair.Draw(past)
		if !drawnAt.Rect.Eq(drawnPast.Rect) || !slices.Equal(drawnAt.Pix, drawnPast.Pix) {
			t.Errorf("%s past its maximum %d draws differently, so the knob stops short", knob.Name, knob.Max)
		}
	}
}

func TestAnUnknownValueIsRefused(t *testing.T) {
	if _, err := crosshair.Default().WithValue("size", 3); !errors.Is(err, crosshair.ErrUnknownKnob) {
		t.Errorf("got %v, want ErrUnknownKnob", err)
	}
}

func TestAPictureIsScaledInQuarterSteps(t *testing.T) {
	if got := crosshair.PictureScale(); got != (crosshair.Knob{Name: "scale", Min: 25, Max: 400, Step: 25}) {
		t.Errorf("got %+v", got)
	}
}
