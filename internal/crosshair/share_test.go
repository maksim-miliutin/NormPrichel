package crosshair_test

import (
	"encoding/json"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

func shared(shape crosshair.Shape, rounding int) crosshair.Style {
	return crosshair.Style{Shape: shape, Color: white, Opacity: 100, Length: 10, Thickness: 6, Gap: 5, Radius: 12, DotSize: 6, Rounding: rounding}
}

func TestAllOfTheWayToRoundDrawsAsRoundCorners(t *testing.T) {
	for _, shape := range roundedShapes {
		round := shared(shape, 0)
		round.Corners = crosshair.Round
		for _, rounding := range []int{100, 250} {
			if !same(draw(t, shared(shape, rounding)), draw(t, round)) {
				t.Errorf("%s rounded %d%% draws otherwise than with round corners", shape, rounding)
			}
		}
	}
}

func TestNoShareOfRoundingLeavesTheCornersAsTheyWere(t *testing.T) {
	for _, shape := range roundedShapes {
		for _, corners := range []crosshair.Corners{crosshair.Sharp, crosshair.Soft, crosshair.Round} {
			style := shared(shape, 0)
			style.Corners = corners
			want := draw(t, style)
			style.Rounding = -20
			if !same(draw(t, style), want) {
				t.Errorf("%s with %s corners draws otherwise when rounded below nothing", shape, corners)
			}
		}
	}
}

func TestAShareOfRoundingOverridesTheKindOfCorners(t *testing.T) {
	for _, shape := range roundedShapes {
		want := draw(t, shared(shape, 40))
		for _, corners := range []crosshair.Corners{crosshair.Soft, crosshair.Round} {
			style := shared(shape, 40)
			style.Corners = corners
			if !same(draw(t, style), want) {
				t.Errorf("%s rounded 40%% draws otherwise with %s corners named too", shape, corners)
			}
		}
	}
}

// The horseshoe gains instead: its caps reach on past the cut as far as they round.
func TestMoreRoundingTakesMoreAway(t *testing.T) {
	for _, shape := range roundedShapes {
		last := area(draw(t, shared(shape, 25)))
		for _, rounding := range []int{50, 75, 100} {
			got := area(draw(t, shared(shape, rounding)))
			if gained := got > last; gained != (shape == crosshair.Horseshoe) || got == last {
				t.Errorf("%s covers %.2f pixels rounded %d%% after %.2f a quarter less", shape, got, rounding, last)
			}
			last = got
		}
	}
}

func TestEveryShareOfRoundingKeepsTheShapeSymmetric(t *testing.T) {
	asymmetric := []crosshair.Shape{crosshair.Tee, crosshair.Chevron, crosshair.Brackets, crosshair.Triangle, crosshair.Horseshoe}
	for _, shape := range roundedShapes {
		for _, rounding := range []int{1, 30, 70, 99} {
			for _, thickness := range []int{3, 4} {
				style := shared(shape, rounding)
				style.Thickness, style.DotSize = thickness, thickness
				img := draw(t, style)
				if !same(img, mirrored(img)) {
					t.Errorf("%s rounded %d%% at thickness %d is not symmetric left to right", shape, rounding, thickness)
				}
				if !slices.Contains(asymmetric, shape) && !same(img, transposed(img)) {
					t.Errorf("%s rounded %d%% at thickness %d is not symmetric along the diagonal", shape, rounding, thickness)
				}
			}
		}
	}
}

func TestAHalfRoundArmRoundsAQuarterOfItsThicknessAtEachCorner(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.Cross, Color: white, Opacity: 100, Length: 12, Thickness: 8, Gap: 8, Rounding: 50}
	want := 4 * (12*8 - (4-math.Pi)*2*2)

	if got := area(draw(t, style)); !near(got, want) {
		t.Errorf("the arms cover %.2f pixels, want %.2f", got, want)
	}
}

// A slight rounding leaves the inner square sharp: its corners curve 2h less than the outer ones.
func TestARoundedSquareIsTheBandBetweenTwoRoundedSquares(t *testing.T) {
	cornerless := func(half, r float64) float64 { return 4*half*half - (4-math.Pi)*r*r }
	for _, c := range []struct {
		rounding     int
		outer, inner float64
	}{
		{25, 1.5, 0},
		{50, 3, 1},
		{100, 6, 4},
	} {
		style := crosshair.Style{Shape: crosshair.Square, Color: white, Opacity: 100, Thickness: 2, Radius: 10, Rounding: c.rounding}
		want := cornerless(11, c.outer) - cornerless(9, c.inner)

		if got := area(draw(t, style)); !near(got, want) {
			t.Errorf("rounded %d%% the square covers %.2f pixels, want %.2f", c.rounding, got, want)
		}
	}
}

func TestAHorseshoeCapReachesPastItsCutAsFarAsItRounds(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.Horseshoe, Color: white, Opacity: 100, Thickness: 8, Radius: 10, DotSize: 4, Rounding: 50}
	h, r := 4.0, 2.0
	ring := 0.75 * math.Pi * 4 * 10 * h
	caps := 4*r*(h-r) + math.Pi*r*r
	dot := 16 - (4-math.Pi)*1*1

	if got := area(draw(t, style)); !nearSlanted(got, ring+caps+dot) {
		t.Errorf("the horseshoe covers %.2f pixels, want %.2f", got, ring+caps+dot)
	}
}

// The top of the brackets runs from x 3 to 10 and y -12 to -8; half way to round, its end at the
// opening loses a pixel's round at each corner, and no more.
func TestBracketEndsRoundAsFarAsTheirCorners(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.Brackets, Color: white, Opacity: 100, Thickness: 4, Gap: 3, Radius: 10, DotSize: 4, Rounding: 50}
	img := draw(t, style)
	reach := img.Bounds().Dx() / 2

	if a := float64(img.RGBAAt(reach+3, reach-12).A); math.Abs(a-255*math.Pi/4) > 4 {
		t.Errorf("the corner of the end is covered %.0f of 255, want a quarter disc of a pixel, %.0f", a, 255*math.Pi/4)
	}
	if a := img.RGBAAt(reach+3, reach-11).A; a != 255 {
		t.Errorf("below the corner the end is covered %d of 255, want fully", a)
	}
}

func TestAShareOfRoundingIsWrittenDownOnlyWhenChosen(t *testing.T) {
	text, err := json.Marshal(crosshair.Default())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(text), "rounding") {
		t.Errorf("no rounding is written down: %s", text)
	}

	style := shared(crosshair.Diamond, 35)
	if text, err = json.Marshal(style); err != nil {
		t.Fatal(err)
	}
	var back crosshair.Style
	if err := json.Unmarshal(text, &back); err != nil {
		t.Fatal(err)
	}
	if back != style {
		t.Errorf("came back as %+v from %s", back, text)
	}
}
