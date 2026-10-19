package crosshair_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

// row reads the cover from the middle out to the right, and column from the middle down.
func row(t *testing.T, style crosshair.Style) (row, column []uint8) {
	t.Helper()
	img := draw(t, style)
	middle := img.Bounds().Dx() / 2

	for i := middle; i < img.Bounds().Dx(); i++ {
		row, column = append(row, img.RGBAAt(i, middle).A), append(column, img.RGBAAt(middle, i).A)
	}

	return row, column
}

func TestAShadowIsWrittenDownOnlyWhenChosen(t *testing.T) {
	text, err := json.Marshal(crosshair.Default())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(text), "shadow") {
		t.Errorf("a plain outline is written down as a shadow: %s", text)
	}

	style := crosshair.Default()
	style.Outline.Shadow = true
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

func TestAShadowFadesOutTwiceAsFarAsTheOutlineReaches(t *testing.T) {
	for _, width := range []int{1, 2, 4} {
		style := crosshair.Style{Shape: crosshair.Dot, Color: white, Opacity: 100, DotSize: 4, Outline: crosshair.Outline{Width: width, Color: black}}
		hard, _ := row(t, style)
		style.Outline.Shadow = true
		soft, down := row(t, style)

		if len(soft) != len(hard)+width {
			t.Errorf("width %d: the shadow reaches %d pixels from the middle and the outline %d, want %d further", width, len(soft), len(hard), width)
		}
		for i := 2; i < len(soft)-1; i++ {
			if soft[i+1] >= soft[i] {
				t.Errorf("width %d: the shadow does not fade %d pixels out: %v", width, i+1, soft)
			}
		}
		if soft[2] == 255 || soft[len(soft)-1] == 0 {
			t.Errorf("width %d: the shadow reads %v, want it fading from the dot to the edge of the drawing", width, soft)
		}
		if string(down) != string(soft) {
			t.Errorf("width %d: the shadow reads %v down and %v across, want it alike all round", width, down, soft)
		}
	}
}

func TestAShadowTakesTheOutlineColourAndLeavesTheFillAlone(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.Cross, Color: white, Opacity: 100, Length: 6, Thickness: 2, Gap: 2, Outline: crosshair.Outline{Width: 2, Color: crosshair.RGB{R: 200, G: 0, B: 0}, Shadow: true}}
	img := draw(t, style)
	middle := img.Bounds().Dx() / 2

	if c := img.RGBAAt(middle+4, middle); c.R != 255 || c.G != 255 || c.A != 255 {
		t.Errorf("an arm shows %v, want its own white", c)
	}
	if c := img.RGBAAt(middle+4, middle+2); c.A == 0 || c.G != 0 || c.R == 0 {
		t.Errorf("the shadow beside an arm shows %v, want a shade of the outline red", c)
	}
}
