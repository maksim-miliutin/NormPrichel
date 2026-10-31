package gamecode_test

import (
	"errors"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/gamecode"
)

func readAt(t *testing.T, code string, height int) crosshair.Style {
	t.Helper()
	s, err := gamecode.Read(code, height)
	if err != nil {
		t.Fatalf("%q: %v", code, err)
	}

	return s
}

// Static cross, no dot, no outline, colour 50 250 50, gap 4, length 8, thickness 2 at 1080 pixels.
const staticCross = "CSGO-hLbCn-69VT6-Bok83-9MOqW-SWzwQ"

func TestACS2CrossIsBuiltOfItsLinesAtItsSize(t *testing.T) {
	s := readAt(t, staticCross, 1080)

	if !s.Built || s.Color != (crosshair.RGB{R: 50, G: 250, B: 50}) || s.Outline.Width != 0 {
		t.Fatalf("got built %v in %v with an outline of %d", s.Built, s.Color, s.Outline.Width)
	}
	if want := lines(true, 100, 8, 2, 4); s.Parts.Inner != want || s.Parts.Outer.On || s.Parts.Dot.On || s.Parts.Ring.On {
		t.Errorf("got parts %+v, want inner lines %+v alone", s.Parts, want)
	}
}

func TestACS2CrosshairGrowsWithTheScreenAsTheGameDrawsIt(t *testing.T) {
	if got, want := readAt(t, staticCross, 2160).Parts.Inner, lines(true, 100, 16, 4, 8); got != want {
		t.Errorf("at twice the height the lines are %+v, want %+v", got, want)
	}
	if got, want := readAt(t, staticCross, 0).Parts.Inner, lines(true, 100, 8, 2, 4); got != want {
		t.Errorf("on a screen of no known height the lines are %+v, want the code's own %+v", got, want)
	}
}

func TestACS2DotStyleShowsTheDotAloneAsThickAsTheLines(t *testing.T) {
	s := readAt(t, "CSGO-MnUCC-89iG7-2cVar-wy7Yn-amCpF", 1080)

	if want := (crosshair.Spot{On: true, Opacity: 100, Size: 3}); s.Parts.Dot != want || s.Parts.Inner.On || s.Parts.Ring.On || s.Outline.Width != 1 {
		t.Errorf("got parts %+v with an outline of %d, want the dot %+v alone and an outline", s.Parts, s.Outline.Width, want)
	}
}

// The circle is a guess: CS2 does not say which sizes draw it, so its inner edge stands at the gap.
func TestACS2CircleIsARingOutsideItsGap(t *testing.T) {
	s := readAt(t, "CSGO-p4NqQ-mV2es-p2UF3-Q9HWe-fVmoE", 1080)

	if want := (crosshair.Band{On: true, Opacity: 100, Radius: 5, Thickness: 1}); s.Parts.Ring != want || s.Parts.Inner.On {
		t.Errorf("got parts %+v, want the ring %+v alone", s.Parts, want)
	}
}

func TestACS2SquareIsTheReadySquareHeldToItsRange(t *testing.T) {
	s := readAt(t, "CSGO-G8oAC-RyvWc-Hi3CZ-voSwn-QJbfE", 768)

	if s.Built || s.Shape != crosshair.SquareDot || s.Thickness != 16 || s.DotSize != 16 || s.Radius != 35 || s.Outline.Width != 1 || s.Color != (crosshair.RGB{R: 124, G: 57, B: 57}) {
		t.Errorf("got %+v, want a square with a dot, 16 thick, 35 out", s)
	}
}

func TestACS2CodeCarriesItsOpacitiesOutlineColourAndTeeStyle(t *testing.T) {
	s := readAt(t, "CSAB6Oby2JUDOrBRy4jRcsR6JOQdhbszKF9GuKQGhKVwAc", 1440)
	want := lines(true, 78, 64, 12, 0)
	want.Hidden.Up = true

	if s.Parts.Inner != want || s.Outline != (crosshair.Outline{Width: 1, Color: crosshair.RGB{R: 40, G: 50, B: 60}}) || s.Parts.OutlineOpacity != 50 || s.Color != (crosshair.RGB{R: 10, G: 20, B: 30}) {
		t.Errorf("got lines %+v, outline %+v at %d%% in %v", s.Parts.Inner, s.Outline, s.Parts.OutlineOpacity, s.Color)
	}
}

func TestAnOldCSGOCodeIsRefusedForWhatItIs(t *testing.T) {
	if _, err := gamecode.Read("CSGO-Cn37R-YE7vo-pLCAL-aURmZ-z6zkG", 1080); !errors.Is(err, gamecode.ErrCSGO) {
		t.Errorf("got %v, want ErrCSGO", err)
	}
	if _, err := gamecode.Read("CShello", 1080); !errors.Is(err, gamecode.ErrUnknown) {
		t.Errorf("a word starting CS gives %v, want ErrUnknown", err)
	}
}

func TestACS2CrossOfNoLengthLeavesItsDotAlone(t *testing.T) {
	s := readAt(t, "CSGO-tsJdC-2PFX6-fw8qD-6GDaX-rYCUM", 768)

	if want := (crosshair.Spot{On: true, Opacity: 100, Size: 3}); s.Parts.Inner.On || s.Parts.Dot != want {
		t.Errorf("got lines %v and dot %+v, want the dot %+v alone", s.Parts.Inner.On, s.Parts.Dot, want)
	}
}
