package window_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/window"
)

var (
	inkColour   = color.RGBA{A: 255}
	paperColour = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

func TestEveryShapeIconIsInkOnPaperInsideItsSquare(t *testing.T) {
	for _, colours := range [][2]color.RGBA{{inkColour, paperColour}, {paperColour, inkColour}} {
		ink, paper := colours[0], colours[1]
		for _, side := range []int{20, 30, 45, 60} {
			for _, shape := range crosshair.Shapes() {
				icon, err := window.ShapeIcon{Shape: shape, Side: side, Ink: ink, Paper: paper}.Image()
				if err != nil {
					t.Fatal(err)
				}
				if inked, edge := inkOf(icon, ink, paper); icon.Rect != image.Rect(0, 0, side, side) || inked == 0 || edge != 0 {
					t.Errorf("%s at %d in %v: %v with %d inked pixels and %d at the edge", shape, side, ink, icon.Rect, inked, edge)
				}
			}
		}
	}
}

// inkOf counts the pixels of the ink colour, and the pixels other than paper on the edge.
func inkOf(icon *image.RGBA, ink, paper color.RGBA) (int, int) {
	inked, edge := 0, 0
	side := icon.Rect.Dx()
	for y := range side {
		for x := range side {
			c := icon.RGBAAt(x, y)
			if c == ink {
				inked++
			}
			if c != paper && (x == 0 || y == 0 || x == side-1 || y == side-1) {
				edge++
			}
		}
	}

	return inked, edge
}

func TestEveryShapeIconLooksDifferent(t *testing.T) {
	seen := map[string]crosshair.Shape{}
	for _, shape := range crosshair.Shapes() {
		icon, err := window.ShapeIcon{Shape: shape, Side: 30, Ink: inkColour, Paper: paperColour}.Image()
		if err != nil {
			t.Fatal(err)
		}

		if other, ok := seen[string(icon.Pix)]; ok {
			t.Errorf("%s looks the same as %s", shape, other)
		}
		seen[string(icon.Pix)] = shape
	}
}

func TestAShapeIconCentresWhatItsInkCovers(t *testing.T) {
	icon, err := window.ShapeIcon{Shape: crosshair.Chevron, Side: 30, Ink: inkColour, Paper: paperColour}.Image()
	if err != nil {
		t.Fatal(err)
	}

	top, bottom := 30, 0
	for y := range 30 {
		for x := range 30 {
			if icon.RGBAAt(x, y) != paperColour {
				top, bottom = min(top, y), max(bottom, y+1)
			}
		}
	}
	if above, below := top, 30-bottom; above-below > 1 || below-above > 1 {
		t.Errorf("the chevron leaves %d rows above and %d below", above, below)
	}
}

func TestAnUnknownShapeHasNoIcon(t *testing.T) {
	if _, err := (window.ShapeIcon{Shape: "star", Side: 30, Ink: inkColour, Paper: paperColour}).Image(); err == nil {
		t.Error("an unknown shape got an icon")
	}
}

func TestAShapeIconTakesTheCornersOfTheCrosshair(t *testing.T) {
	for _, shape := range []crosshair.Shape{crosshair.Cross, crosshair.Square, crosshair.Circle} {
		look := window.ShapeIcon{Shape: shape, Side: 34, Ink: inkColour, Paper: paperColour}
		sharp, err := look.Image()
		if err != nil {
			t.Fatal(err)
		}
		look.Corners = crosshair.Round
		round, err := look.Image()
		if err != nil {
			t.Fatal(err)
		}

		if changed := string(round.Pix) != string(sharp.Pix); changed != crosshair.Rounds(shape) {
			t.Errorf("%s: the round icon differs from the sharp one %v, want %v", shape, changed, crosshair.Rounds(shape))
		}
	}
}
