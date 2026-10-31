package crosshair_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

func TestEveryPartOffersItsOwnKnobs(t *testing.T) {
	want := map[crosshair.Part][]string{
		crosshair.InnerPart:   {"opacity", "length", "thickness", "gap"},
		crosshair.OuterPart:   {"opacity", "length", "thickness", "gap"},
		crosshair.DotPart:     {"opacity", "dot"},
		crosshair.RingPart:    {"opacity", "radius", "thickness"},
		crosshair.OutlinePart: {"outline", "outlineOpacity"},
	}
	for _, part := range crosshair.PartNames() {
		var got []string
		for _, k := range crosshair.PartKnobs(part) {
			got = append(got, k.Name)
			if k.Step != 1 || k.Max <= k.Min {
				t.Errorf("%s %s runs %+v", part, k.Name, k)
			}
		}
		if !slices.Equal(got, want[part]) {
			t.Errorf("%s offers %v, want %v", part, got, want[part])
		}
	}
	if len(want) != len(crosshair.PartNames()) || crosshair.PartKnobs("tail") != nil {
		t.Errorf("the parts are %v", crosshair.PartNames())
	}
}

// Each knob is set to a value of its own, so a knob that reached another one's field shows.
func TestEachPartKnobChangesItsOwnValueAlone(t *testing.T) {
	var style crosshair.Style
	set := map[string]int{}
	for _, part := range crosshair.PartNames() {
		for _, k := range crosshair.PartKnobs(part) {
			value := k.Min + 1 + len(set)%(k.Max-k.Min)
			changed, err := style.WithPartValue(part, k.Name, value)
			if err != nil {
				t.Fatal(err)
			}
			style = changed
			set[string(part)+"."+k.Name] = value
		}
	}

	for _, part := range crosshair.PartNames() {
		for _, k := range crosshair.PartKnobs(part) {
			if got, want := style.PartValue(part, k.Name), set[string(part)+"."+k.Name]; got != want {
				t.Errorf("%s %s reads %d, want %d", part, k.Name, got, want)
			}
		}
	}
	if style.Outline.Width != set["outline.outline"] {
		t.Errorf("the outline knob of the parts sets the width %d, want %d", style.Outline.Width, set["outline.outline"])
	}
}

func TestAPartKnobIsHeldToItsRangeAndAnUnknownOneRefused(t *testing.T) {
	style, err := crosshair.Style{}.WithPartValue(crosshair.InnerPart, "length", 99)
	if err != nil || style.PartValue(crosshair.InnerPart, "length") != 64 {
		t.Errorf("a length of 99 holds %d, %v; want 64", style.PartValue(crosshair.InnerPart, "length"), err)
	}
	if _, err := style.WithPartValue(crosshair.DotPart, "length", 3); !errors.Is(err, crosshair.ErrUnknownKnob) {
		t.Errorf("the dot's length gives %v, want ErrUnknownKnob", err)
	}
}

func TestEveryPartButTheOutlineTurnsOnAndOffAlone(t *testing.T) {
	for _, part := range crosshair.PartNames() {
		style := crosshair.Style{}.WithPartShown(part, true)
		if got, want := style.PartShown(part), part != crosshair.OutlinePart; got != want || crosshair.Switched(part) != want {
			t.Errorf("%s turned on shows %v and has a switch %v, want %v", part, got, crosshair.Switched(part), want)
		}
		for _, other := range crosshair.PartNames() {
			if other != part && style.PartShown(other) {
				t.Errorf("turning %s on turned %s on too", part, other)
			}
		}
		if style.WithPartShown(part, false).PartShown(part) {
			t.Errorf("%s stays on when turned off", part)
		}
	}
}

func TestEachArmOfEitherLinesHidesAlone(t *testing.T) {
	for _, part := range crosshair.PartNames() {
		lines := part == crosshair.InnerPart || part == crosshair.OuterPart
		if crosshair.Armed(part) != lines {
			t.Errorf("%s has arms %v, want %v", part, crosshair.Armed(part), lines)
		}
		for _, arm := range crosshair.ArmNames() {
			style := crosshair.Style{}.WithArm(part, arm, false)
			for _, other := range crosshair.PartNames() {
				for _, each := range crosshair.ArmNames() {
					want := (other == crosshair.InnerPart || other == crosshair.OuterPart) && (other != part || each != arm)
					if got := style.ArmShown(other, each); got != want {
						t.Errorf("after hiding %s %s, %s %s shows %v, want %v", part, arm, other, each, got, want)
					}
				}
			}
			if lines && !style.WithArm(part, arm, true).ArmShown(part, arm) {
				t.Errorf("%s %s stays hidden when shown again", part, arm)
			}
		}
	}
}

// A ring's border is counted more exactly built than ready, so the rings go without an outline.
func TestBuildingAShapePartsCanDrawKeepsItsLook(t *testing.T) {
	for _, shape := range []crosshair.Shape{crosshair.Dot, crosshair.Cross, crosshair.CrossDot, crosshair.Tee, crosshair.Dots, crosshair.Circle, crosshair.CircleDot, crosshair.CircleCross} {
		for _, opacity := range []int{100, 70} {
			for _, rounding := range []int{0, 50} {
				style := crosshair.Style{Shape: shape, Color: white, Opacity: opacity, Length: 7, Thickness: 3, Gap: 4, Radius: 11, DotSize: 3, Rounding: rounding, Outline: crosshair.Outline{Width: 1, Color: black}}
				if slices.Contains([]crosshair.Shape{crosshair.Circle, crosshair.CircleDot, crosshair.CircleCross}, shape) {
					style.Outline.Width = 0
				}
				if !same(draw(t, style.Build()), draw(t, style)) {
					t.Errorf("%s at %d%% rounded %d%% built from parts draws otherwise", shape, opacity, rounding)
				}
			}
		}
	}
}

func TestBuildingAShapePartsCannotDrawStartsFromItsSizes(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.X, Color: white, Opacity: 80, Length: 7, Thickness: 3, Gap: 2, Radius: 11, DotSize: 4}.Build()
	want := crosshair.Parts{
		Inner:          crosshair.Lines{On: true, Opacity: 80, Length: 7, Thickness: 3, Gap: 2},
		Outer:          crosshair.Lines{Opacity: 80, Length: 4, Thickness: 3, Gap: 13},
		Dot:            crosshair.Spot{On: true, Opacity: 80, Size: 4},
		Ring:           crosshair.Band{Opacity: 80, Radius: 11, Thickness: 3},
		OutlineOpacity: 80,
	}

	if !style.Built || style.Shape != crosshair.X || style.Parts != want {
		t.Errorf("built %v from %s with parts %+v, want %+v", style.Built, style.Shape, style.Parts, want)
	}
}

func TestBuildingAgainFindsThePartsAsTheyWereLeft(t *testing.T) {
	style := crosshair.Default().Build()
	style.Parts.Outer.On = true
	style.Parts.Inner.Hidden.Down = true
	left := style.Parts
	style.Built = false
	style.Length = 30

	if again := style.Build(); again.Parts != left {
		t.Errorf("built again the parts are %+v, want them as left, %+v", again.Parts, left)
	}
}
