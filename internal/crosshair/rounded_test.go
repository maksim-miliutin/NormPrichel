package crosshair_test

import (
	"encoding/json"
	"image"
	"math"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

var roundedShapes = []crosshair.Shape{crosshair.Dot, crosshair.Cross, crosshair.CrossDot, crosshair.Tee, crosshair.Dots, crosshair.CircleDot, crosshair.CircleCross}

func draw(t *testing.T, style crosshair.Style) *image.RGBA {
	t.Helper()
	img, err := crosshair.Draw(style)
	if err != nil {
		t.Fatal(err)
	}

	return img
}

// area counts the pixels a drawing covers, a half covered pixel as a half.
func area(img *image.RGBA) float64 {
	sum := 0
	for i := 3; i < len(img.Pix); i += 4 {
		sum += int(img.Pix[i])
	}

	return float64(sum) / 255
}

func near(got, want float64) bool {
	return math.Abs(got-want) <= 0.01*want
}

func TestCornersStaySharpUnlessChosenAndKeepOutOfOldFiles(t *testing.T) {
	if crosshair.Default().Corners != crosshair.Sharp {
		t.Errorf("the default crosshair has %q corners, want sharp", crosshair.Default().Corners)
	}
	text, err := json.Marshal(crosshair.Default())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(text), "corners") {
		t.Errorf("sharp corners are written down: %s", text)
	}

	style := crosshair.Default()
	style.Corners = crosshair.Round
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

func TestCornersOfAnUnknownKindDrawSharp(t *testing.T) {
	style := crosshair.Default()
	style.Corners = "squircle"

	if !same(draw(t, style), draw(t, crosshair.Default())) {
		t.Error("unknown corners draw differently from sharp ones")
	}
}

func TestRoundingOnlyEverTakesAwayFromTheSharpShape(t *testing.T) {
	for _, shape := range roundedShapes {
		for i, style := range styles(shape) {
			if i%23 != 0 {
				continue
			}
			sharp := draw(t, style)
			for _, corners := range []crosshair.Corners{crosshair.Soft, crosshair.Round} {
				style.Corners = corners
				rounded := draw(t, style)
				if rounded.Bounds() != sharp.Bounds() {
					t.Fatalf("%+v is %v, want the size of the sharp %v", style, rounded.Bounds(), sharp.Bounds())
				}
				for p := 3; p < len(sharp.Pix); p += 4 {
					if rounded.Pix[p] > sharp.Pix[p] {
						t.Errorf("%+v covers pixel %d more than the sharp shape does", style, p/4)
						break
					}
				}
			}
		}
	}
}

func TestARoundArmIsACapsuleOfItsLengthAndThickness(t *testing.T) {
	for _, thickness := range []int{2, 3, 4, 6} {
		style := crosshair.Style{Shape: crosshair.Cross, Color: white, Opacity: 100, Length: 12, Thickness: thickness, Gap: thickness, Corners: crosshair.Round}
		w := float64(thickness)
		want := 4 * ((12-w)*w + math.Pi*w*w/4)

		if got := area(draw(t, style)); !near(got, want) {
			t.Errorf("thickness %d: the arms cover %.2f pixels, want %.2f", thickness, got, want)
		}
	}
}

func TestSoftCornersRoundOffAPixelAtEachCorner(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.Cross, Color: white, Opacity: 100, Length: 12, Thickness: 4, Gap: 4, Corners: crosshair.Soft}
	want := 4 * (12*4 - 4*(1-math.Pi/4))

	if got := area(draw(t, style)); !near(got, want) {
		t.Errorf("the arms cover %.2f pixels, want %.2f", got, want)
	}
}

func TestARoundDotIsADiscAndItsOutlineARingAroundIt(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.Dot, Color: white, Opacity: 100, DotSize: 8, Corners: crosshair.Round}
	if got, want := area(draw(t, style)), math.Pi*16; !near(got, want) {
		t.Errorf("the dot covers %.2f pixels, want %.2f", got, want)
	}

	style.Outline = crosshair.Outline{Width: 2, Color: black}
	if got, want := area(draw(t, style)), math.Pi*36; !near(got, want) {
		t.Errorf("the dot with its outline covers %.2f pixels, want %.2f", got, want)
	}
}

func TestRoundedShapesMirrorAsTheSharpOnesDo(t *testing.T) {
	for _, shape := range roundedShapes {
		for i, style := range styles(shape) {
			if i%17 != 0 || style.DotSize%2 != style.Thickness%2 {
				continue
			}
			for _, corners := range []crosshair.Corners{crosshair.Soft, crosshair.Round} {
				style.Corners = corners
				img := draw(t, style)
				if !same(img, mirrored(img)) {
					t.Errorf("%+v is not symmetric left to right", style)
				}
				if shape != crosshair.Tee && !same(img, transposed(img)) {
					t.Errorf("%+v is not symmetric along the diagonal", style)
				}
			}
		}
	}
}

func TestEveryRoundedShapeLosesItsCorners(t *testing.T) {
	for _, shape := range roundedShapes {
		style := crosshair.Style{Shape: shape, Color: white, Opacity: 100, Length: 8, Thickness: 4, Gap: 4, Radius: 9, DotSize: 4}
		sharp := draw(t, style)
		for _, corners := range []crosshair.Corners{crosshair.Soft, crosshair.Round} {
			style.Corners = corners
			if got := area(draw(t, style)); got >= area(sharp) {
				t.Errorf("%s with %s corners covers %.2f pixels, want fewer than the sharp %.2f", shape, corners, got, area(sharp))
			}
		}
	}
}
