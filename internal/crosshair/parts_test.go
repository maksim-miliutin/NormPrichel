package crosshair_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

func built(parts crosshair.Parts) crosshair.Style {
	return crosshair.Style{Shape: crosshair.Cross, Color: white, Opacity: 100, Built: true, Parts: parts}
}

func lines(length, thickness, gap int) crosshair.Lines {
	return crosshair.Lines{On: true, Opacity: 100, Length: length, Thickness: thickness, Gap: gap}
}

func TestPartsAreDrawnToTheirPixels(t *testing.T) {
	tee := lines(2, 1, 1)
	tee.Hidden.Up = true
	for _, c := range []struct {
		name  string
		parts crosshair.Parts
		want  []string
	}{
		{
			name:  "inner lines alone are a cross",
			parts: crosshair.Parts{Inner: lines(2, 1, 1)},
			want:  []string{"..#..", "..#..", "##.##", "..#..", "..#.."},
		},
		{
			name:  "a hidden arm leaves the frame of the others",
			parts: crosshair.Parts{Inner: tee},
			want:  []string{".....", ".....", "##.##", "..#..", "..#.."},
		},
		{
			name:  "a dot of the other parity gives its spare half pixel right and down",
			parts: crosshair.Parts{Inner: lines(1, 1, 2), Dot: crosshair.Spot{On: true, Opacity: 100, Size: 2}},
			want:  []string{"..#..", ".....", "#.###", "..##.", "..#.."},
		},
		{
			name:  "outer lines stand past the inner ones",
			parts: crosshair.Parts{Inner: lines(1, 1, 1), Outer: lines(1, 1, 3)},
			want:  []string{"...#...", ".......", "...#...", "#.#.#.#", "...#...", ".......", "...#..."},
		},
	} {
		if got := art(draw(t, built(c.parts))); !slices.Equal(got, c.want) {
			t.Errorf("%s:\n%s\nwant\n%s", c.name, strings.Join(got, "\n"), strings.Join(c.want, "\n"))
		}
	}
}

// A clear line keeps its outline: the trick games know for a hollow crosshair.
func TestAClearPartShowsOnlyItsOutline(t *testing.T) {
	clear := lines(2, 1, 1)
	clear.Opacity = 0
	style := built(crosshair.Parts{Inner: clear, OutlineOpacity: 100})
	style.Outline = crosshair.Outline{Width: 1, Color: black}
	want := []string{"..ooo..", "..o.o..", "ooo.ooo", "o..o..o", "ooo.ooo", "..o.o..", "..ooo.."}

	if got := art(draw(t, style)); !slices.Equal(got, want) {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestEachPartLaysItsOwnOpacityOverTheOthers(t *testing.T) {
	half := lines(2, 2, 0)
	half.Opacity = 50
	img := draw(t, built(crosshair.Parts{Inner: half, Dot: crosshair.Spot{On: true, Opacity: 60, Size: 2}}))
	middle := img.Bounds().Dx() / 2

	if c := img.RGBAAt(middle, middle); c.A != 204 || c.R != 204 {
		t.Errorf("under the line and the dot the colour is %v, want 80%% of white", c)
	}
	if c := img.RGBAAt(middle+1, middle-1); c.A != 128 || c.R != 128 {
		t.Errorf("under the line alone the colour is %v, want half of white", c)
	}
}

func TestPartsOfOneOpacityDoNotDarkenWhereTheyCross(t *testing.T) {
	half := lines(2, 2, 0)
	half.Opacity = 50
	img := draw(t, built(crosshair.Parts{Inner: half, Dot: crosshair.Spot{On: true, Opacity: 50, Size: 2}}))
	middle := img.Bounds().Dx() / 2

	if c := img.RGBAAt(middle, middle); c.A != 128 || c.R != 128 {
		t.Errorf("under the line and the dot the colour is %v, want half of white", c)
	}
}

func TestTheOutlineOfPartsTakesItsOwnOpacity(t *testing.T) {
	style := built(crosshair.Parts{Inner: lines(2, 1, 1), OutlineOpacity: 50})
	style.Outline = crosshair.Outline{Width: 1, Color: black}
	img := draw(t, style)

	if c := img.RGBAAt(4, 2); c.A != 128 || c.R != 0 {
		t.Errorf("beside the line the outline is %v, want half of black", c)
	}
	if c := img.RGBAAt(3, 1); c.A != 255 || c.R != 255 {
		t.Errorf("the line is %v, want white", c)
	}
}

// Where a ring crosses a line two parts cover one pixel in part, and only counting their samples
// together keeps it as solid as the ready shape. The ring goes without an outline: its edge, grown
// a pixel, may not cover a pixel the ring cuts, and the ready shape counts that border less exactly.
func TestLinesADotAndARingBuildTheReadyShapesToTheBit(t *testing.T) {
	tee := lines(8, 2, 3)
	tee.Hidden.Up = true
	for _, c := range []struct {
		shape   crosshair.Shape
		outline int
		parts   crosshair.Parts
	}{
		{crosshair.CrossDot, 1, crosshair.Parts{Inner: lines(8, 2, 3), Dot: crosshair.Spot{On: true, Opacity: 100, Size: 2}, OutlineOpacity: 100}},
		{crosshair.Tee, 1, crosshair.Parts{Inner: tee, OutlineOpacity: 100}},
		{crosshair.CircleCross, 0, crosshair.Parts{Inner: lines(8, 2, 3), Ring: crosshair.Band{On: true, Opacity: 100, Radius: 10, Thickness: 2}, OutlineOpacity: 100}},
	} {
		for _, rounding := range []int{0, 40, 100} {
			ready := crosshair.Default()
			ready.Shape, ready.Rounding, ready.Outline.Width = c.shape, rounding, c.outline
			assembled := ready
			assembled.Built, assembled.Parts = true, c.parts

			if !same(draw(t, assembled), draw(t, ready)) {
				t.Errorf("%s rounded %d%% built from parts draws otherwise than the ready one", c.shape, rounding)
			}
		}
	}
}

func TestARingWithItsOutlineBuildsTheReadyOneToARoundingStep(t *testing.T) {
	for _, rounding := range []int{0, 100} {
		ready := crosshair.Default()
		ready.Shape, ready.Rounding = crosshair.CircleCross, rounding
		assembled := ready
		assembled.Built = true
		assembled.Parts = crosshair.Parts{Inner: lines(8, 2, 3), Ring: crosshair.Band{On: true, Opacity: 100, Radius: 10, Thickness: 2}, OutlineOpacity: 100}
		a, b := draw(t, assembled), draw(t, ready)

		for i := range a.Pix {
			if d := int(a.Pix[i]) - int(b.Pix[i]); d > 1 || d < -1 {
				t.Errorf("rounded %d%%: byte %d is %d built and %d ready, want a step apart at most", rounding, i, a.Pix[i], b.Pix[i])
				break
			}
		}
	}
}

func TestPartsAreLeftAloneUntilTheCrosshairIsBuilt(t *testing.T) {
	style := crosshair.Default()
	style.Parts = crosshair.Parts{Outer: lines(30, 9, 20)}

	if !same(draw(t, style), draw(t, crosshair.Default())) {
		t.Error("parts change a crosshair drawn from its shape")
	}
}

func TestACrosshairWithNoPartShownDrawsNothing(t *testing.T) {
	img := draw(t, built(crosshair.Parts{Inner: crosshair.Lines{Opacity: 100, Length: 8, Thickness: 2}, Ring: crosshair.Band{Thickness: 2}}))

	if img.Bounds().Empty() || area(img) != 0 {
		t.Errorf("got %v covering %.2f pixels, want a clear picture", img.Bounds(), area(img))
	}
}

func TestPartsPastTheirRangeAreHeldToIt(t *testing.T) {
	wild := built(crosshair.Parts{Inner: crosshair.Lines{On: true, Opacity: 300, Length: 90, Thickness: 40, Gap: -5}})
	tame := built(crosshair.Parts{Inner: crosshair.Lines{On: true, Opacity: 100, Length: 64, Thickness: 16, Gap: 0}})

	if !same(draw(t, wild), draw(t, tame)) {
		t.Error("parts past their range draw otherwise than at its ends")
	}
}

func TestPartsOfOneParityStaySymmetric(t *testing.T) {
	for _, thickness := range []int{1, 2, 3} {
		for _, rounding := range []int{0, 60} {
			parts := crosshair.Parts{
				Inner: lines(4, thickness, 2), Outer: lines(3, thickness, 9),
				Dot:  crosshair.Spot{On: true, Opacity: 70, Size: thickness},
				Ring: crosshair.Band{On: true, Opacity: 40, Radius: 7, Thickness: thickness}, OutlineOpacity: 80,
			}
			style := built(parts)
			style.Rounding, style.Outline = rounding, crosshair.Outline{Width: 1, Color: black}
			img := draw(t, style)

			if !same(img, mirrored(img)) || !same(img, transposed(img)) {
				t.Errorf("thickness %d rounded %d%%: the parts are not symmetric", thickness, rounding)
			}
		}
	}
}

func TestPartsAreWrittenDownOnlyForABuiltCrosshair(t *testing.T) {
	text, err := json.Marshal(crosshair.Default())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(text), "parts") || strings.Contains(string(text), "built") {
		t.Errorf("a crosshair drawn from its shape writes its parts: %s", text)
	}

	style := built(crosshair.Parts{Inner: lines(5, 3, 2), Dot: crosshair.Spot{On: true, Opacity: 40, Size: 3}, OutlineOpacity: 60})
	style.Parts.Inner.Hidden.Left = true
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
