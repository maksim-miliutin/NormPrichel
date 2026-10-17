package window_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/window"
)

var (
	backColour = crosshair.RGB{R: 0x10, G: 0x10, B: 0x10}
	edgeColour = crosshair.RGB{R: 0x80, G: 0x80, B: 0x80}
	fillColour = crosshair.RGB{R: 0xF0, G: 0xA0, B: 0x30}
)

func opaque(c crosshair.RGB) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: 0xFF}
}

func at(img *image.RGBA, x, y int) crosshair.RGB {
	c := img.RGBAAt(x, y)
	return crosshair.RGB{R: c.R, G: c.G, B: c.B}
}

func TestAPlateIsFilledInsideItsEdgeAndLeavesItsCornersToTheBackground(t *testing.T) {
	img := window.Plate{Size: image.Pt(40, 20), Radius: 5, Fill: fillColour, Edge: edgeColour, Width: 1, Back: backColour}.Image()

	for _, c := range []struct {
		x, y int
		want crosshair.RGB
		what string
	}{
		{20, 10, fillColour, "the middle"},
		{20, 0, edgeColour, "the top edge"},
		{39, 10, edgeColour, "the right edge"},
		{0, 0, backColour, "the corner outside the rounding"},
		{39, 19, backColour, "the opposite corner"},
	} {
		if got := at(img, c.x, c.y); got != c.want {
			t.Errorf("%s (%d, %d) is %v, want %v", c.what, c.x, c.y, got, c.want)
		}
	}
}

func TestTheRoundedCornerFadesIntoTheBackgroundRatherThanStepping(t *testing.T) {
	img := window.Plate{Size: image.Pt(40, 20), Radius: 8, Fill: fillColour, Back: backColour}.Image()

	got := at(img, 2, 2)
	if got == backColour || got == fillColour {
		t.Errorf("a pixel the arc cuts through is %v, want a blend of the fill and the background", got)
	}
	if got.R <= backColour.R || got.R >= fillColour.R {
		t.Errorf("the blend %v lies outside the fill and the background", got)
	}
}

func TestAPlateWithoutAnEdgeIsFillRightUpToItsBorder(t *testing.T) {
	img := window.Plate{Size: image.Pt(30, 12), Radius: 6, Fill: fillColour, Back: backColour}.Image()

	if got := at(img, 15, 0); got != fillColour {
		t.Errorf("the top middle is %v, want the fill", got)
	}
	if got := at(img, 0, 0); got != backColour {
		t.Errorf("the corner of a pill is %v, want the background", got)
	}
}

func TestAPictureInAPlateKeepsItsMiddleAndGivesUpItsCorners(t *testing.T) {
	picture := image.NewRGBA(image.Rect(0, 0, 24, 24))
	for y := range 24 {
		for x := range 24 {
			picture.Set(x, y, opaque(fillColour))
		}
	}
	framed := window.Plate{Radius: 6, Edge: edgeColour, Width: 1, Back: backColour}.Around(picture)

	for _, c := range []struct {
		x, y int
		want crosshair.RGB
		what string
	}{
		{12, 12, fillColour, "the middle"},
		{12, 0, edgeColour, "the top edge"},
		{0, 0, backColour, "the corner"},
	} {
		if got := at(framed, c.x, c.y); got != c.want {
			t.Errorf("%s (%d, %d) is %v, want %v", c.what, c.x, c.y, got, c.want)
		}
	}
}
