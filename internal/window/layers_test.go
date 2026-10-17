package window_test

import (
	"image"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/window"
)

func TestALayerCoversTheOnesUnderItAndTheBackShowsElsewhere(t *testing.T) {
	img := window.Layered(image.Pt(30, 20), backColour,
		window.Layer{Rect: image.Rect(2, 2, 28, 18), Colour: edgeColour},
		window.Layer{Rect: image.Rect(6, 6, 24, 14), Colour: fillColour},
	)

	for _, c := range []struct {
		x, y int
		want crosshair.RGB
		what string
	}{
		{10, 10, fillColour, "the top layer"},
		{3, 10, edgeColour, "the layer under it"},
		{0, 0, backColour, "the corner outside both"},
		{29, 19, backColour, "the opposite corner"},
	} {
		if got := at(img, c.x, c.y); got != c.want {
			t.Errorf("%s (%d, %d) is %v, want %v", c.what, c.x, c.y, got, c.want)
		}
	}
}

func TestADiscOverARailShadesItsEdgeIntoTheRailRatherThanTheBack(t *testing.T) {
	rail := image.Rect(0, 8, 20, 12)
	img := window.Layered(image.Pt(20, 20), backColour,
		window.Layer{Rect: rail, Colour: edgeColour},
		window.Layer{Rect: image.Rect(4, 4, 16, 16), Radius: 6, Colour: fillColour},
	)

	blends := neither(img, rail, edgeColour, fillColour)
	if len(blends) == 0 {
		t.Fatal("the edge of the disc cuts no pixel of the rail")
	}
	for _, got := range blends {
		if !between(got, edgeColour, fillColour) {
			t.Errorf("a pixel the edge of the disc cuts is %v, want a blend of the rail %v and the disc %v", got, edgeColour, fillColour)
		}
	}
}

func neither(img *image.RGBA, area image.Rectangle, one, other crosshair.RGB) []crosshair.RGB {
	var found []crosshair.RGB
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			if got := at(img, x, y); got != one && got != other {
				found = append(found, got)
			}
		}
	}

	return found
}

func between(c, one, other crosshair.RGB) bool {
	inside := func(v, a, b uint8) bool { return min(a, b) <= v && v <= max(a, b) }
	return inside(c.R, one.R, other.R) && inside(c.G, one.G, other.G) && inside(c.B, one.B, other.B)
}
