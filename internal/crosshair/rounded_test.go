package crosshair_test

import (
	"encoding/json"
	"image"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

var (
	boxShapes     = []crosshair.Shape{crosshair.Dot, crosshair.Cross, crosshair.CrossDot, crosshair.Tee, crosshair.Dots, crosshair.CircleDot, crosshair.CircleCross}
	frameShapes   = []crosshair.Shape{crosshair.Square, crosshair.SquareDot, crosshair.Brackets, crosshair.Viewfinder}
	roundedShapes = slices.Concat(boxShapes, frameShapes, []crosshair.Shape{crosshair.X, crosshair.XDot, crosshair.Chevron})
)

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

// Across a slant the square grid of samples is coarser, so a slanted edge is counted less finely.
func nearSlanted(got, want float64) bool {
	return math.Abs(got-want) <= 0.02*want
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
	for _, shape := range slices.Concat(boxShapes, frameShapes) {
		for i, style := range styles(shape) {
			if i%23 != 0 {
				continue
			}
			sharp := draw(t, style)
			for _, corners := range []crosshair.Corners{crosshair.Soft, crosshair.Round} {
				if corners == crosshair.Round && slices.Contains(frameShapes, shape) {
					continue
				}
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
				if !slices.Contains([]crosshair.Shape{crosshair.Tee, crosshair.Chevron, crosshair.Brackets}, shape) && !same(img, transposed(img)) {
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

func TestARoundDiagonalArmIsASmoothCapsuleWithinTheSharpSize(t *testing.T) {
	for _, thickness := range []int{2, 3, 4} {
		style := crosshair.Style{Shape: crosshair.X, Color: white, Opacity: 100, Length: 10, Thickness: thickness, Gap: 4}
		sharp := draw(t, style)
		style.Corners = crosshair.Round
		round := draw(t, style)
		w := float64(thickness)
		want := 4 * (9*math.Sqrt2*w + math.Pi*w*w/4)

		if got := area(round); !nearSlanted(got, want) {
			t.Errorf("thickness %d: the arms cover %.2f pixels, want %.2f", thickness, got, want)
		}
		if round.Bounds() != sharp.Bounds() {
			t.Errorf("thickness %d: the round cross is %v, want the size of the sharp %v", thickness, round.Bounds(), sharp.Bounds())
		}
	}
}

func TestASoftDiagonalArmIsABoxWithItsCornersRoundedOff(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.X, Color: white, Opacity: 100, Length: 10, Thickness: 4, Gap: 4, Corners: crosshair.Soft}
	want := 4 * ((9*math.Sqrt2+4)*4 - 4*(1-math.Pi/4))

	if got := area(draw(t, style)); !nearSlanted(got, want) {
		t.Errorf("the arms cover %.2f pixels, want %.2f", got, want)
	}
}

func TestTheArmsOfARoundChevronMeetInARoundTip(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.Chevron, Color: white, Opacity: 100, Length: 8, Thickness: 4, Gap: 3, Corners: crosshair.Round}
	img := draw(t, style)
	middle := img.Bounds().Dx() / 2

	if a := img.RGBAAt(middle, middle).A; a != 255 {
		t.Errorf("the tip is covered %d of 255, want fully", a)
	}
	above := img.RGBAAt(middle, middle-2).A
	if above == 0 || above == 255 {
		t.Errorf("past the tip the cover is %d of 255, want the edge of a round cap", above)
	}
}

func TestARoundDiagonalCrossKeepsItsDotAndTheBareOneItsHole(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.XDot, Color: white, Opacity: 100, Length: 8, Thickness: 4, Gap: 4, DotSize: 4, Corners: crosshair.Round}
	dotted := draw(t, style)
	style.Shape = crosshair.X
	bare := draw(t, style)
	middle := dotted.Bounds().Dx() / 2

	if a := dotted.RGBAAt(middle, middle).A; a != 255 {
		t.Errorf("the middle of the cross with a dot is covered %d of 255, want fully", a)
	}
	if a := bare.RGBAAt(middle, middle).A; a != 0 {
		t.Errorf("the middle of the bare cross is covered %d of 255, want clear", a)
	}
}

func TestARoundSquareIsABandAroundASquareWithHalfRoundCorners(t *testing.T) {
	for _, thickness := range []int{2, 4} {
		style := crosshair.Style{Shape: crosshair.Square, Color: white, Opacity: 100, Thickness: thickness, Radius: 10, Corners: crosshair.Round}
		corner := 5.0
		want := (8*(10-corner) + 2*math.Pi*corner) * float64(thickness)

		if got := area(draw(t, style)); !near(got, want) {
			t.Errorf("thickness %d: the square covers %.2f pixels, want %.2f", thickness, got, want)
		}
	}
}

func TestTheOutlineOfARoundSquareWidensItsBandBothWays(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.Square, Color: white, Opacity: 100, Thickness: 2, Radius: 10, Corners: crosshair.Round, Outline: crosshair.Outline{Width: 2, Color: black}}
	want := (8*(10-5) + 2*math.Pi*5) * (2 + 2*2)

	if got := area(draw(t, style)); !near(got, want) {
		t.Errorf("the square and its outline cover %.2f pixels, want %.2f", got, want)
	}
}

func TestRoundBracketsOpenAsWideAsTheSharpOnesAndCapTheEnds(t *testing.T) {
	for _, thickness := range []int{2, 3} {
		style := crosshair.Style{Shape: crosshair.Brackets, Color: white, Opacity: 100, Thickness: thickness, Gap: 3, Radius: 10, DotSize: thickness}
		sharp := draw(t, style)
		style.Corners = crosshair.Round
		round := draw(t, style)
		parity := thickness % 2
		reach := (sharp.Bounds().Dx() - parity) / 2
		top := reach - 10

		for _, dx := range []int{2, parity - 3} {
			if a := round.RGBAAt(reach+dx, top).A; a != 0 {
				t.Errorf("thickness %d: the opening is covered %d of 255 at %d, want clear", thickness, a, dx)
			}
		}
		for _, dx := range []int{3, parity - 4} {
			edge, capped := sharp.RGBAAt(reach+dx, top).A, round.RGBAAt(reach+dx, top).A
			if edge != 255 || capped == 0 || capped == 255 {
				t.Errorf("thickness %d: at %d the sharp top is covered %d and the round %d, want full and the edge of a cap", thickness, dx, edge, capped)
			}
		}
	}
}

func TestAWideOpeningTightensTheBendSoTheCapSitsOnTheStraight(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.Brackets, Color: white, Opacity: 100, Thickness: 2, Gap: 8, Radius: 12, DotSize: 2, Corners: crosshair.Round}
	img := draw(t, style)
	reach := img.Bounds().Dx() / 2

	if a := img.RGBAAt(reach+10, reach-13).A; a < 128 {
		t.Errorf("past the cap the top edge is covered %d of 255, want the band still running straight there", a)
	}
}

// differences lists the points, counted from the middle pixel, inked in one drawing and bare in the other.
func differences(a, b *image.RGBA, points []image.Point) []image.Point {
	var differ []image.Point
	middle := a.Bounds().Dx() / 2
	for _, p := range points {
		if (a.RGBAAt(middle+p.X, middle+p.Y).A == 0) != (b.RGBAAt(middle+p.X, middle+p.Y).A == 0) {
			differ = append(differ, p)
		}
	}

	return differ
}

func TestARoundedFrameKeepsItsDotAndSidesWhereTheSharpOneHasThem(t *testing.T) {
	middleAndSides := []image.Point{{0, 0}, {-10, 0}, {10, 0}, {0, -10}, {0, 10}}
	for _, shape := range frameShapes {
		style := crosshair.Style{Shape: shape, Color: white, Opacity: 100, Thickness: 3, Gap: 3, Radius: 10, DotSize: 3}
		sharp := draw(t, style)
		for _, corners := range []crosshair.Corners{crosshair.Soft, crosshair.Round} {
			style.Corners = corners
			if differ := differences(sharp, draw(t, style), middleAndSides); len(differ) > 0 {
				t.Errorf("%s with %s corners is inked otherwise than the sharp one at %v", shape, corners, differ)
			}
		}
	}
}
