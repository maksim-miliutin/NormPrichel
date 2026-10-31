package gamecode_test

import (
	"errors"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/gamecode"
)

func read(t *testing.T, code string) crosshair.Style {
	t.Helper()
	s, err := gamecode.Read(code)
	if err != nil {
		t.Fatalf("%q: %v", code, err)
	}

	return s
}

func lines(on bool, opacity, length, thickness, gap int) crosshair.Lines {
	return crosshair.Lines{On: on, Opacity: opacity, Length: length, Thickness: thickness, Gap: gap}
}

// The values Valorant starts with: white, outlines of a pixel at half opacity, no dot, inner
// lines 6 long, 2 thick, 3 out at 0.8, outer lines 2 long, 2 thick, 10 out at 0.35.
func TestTheBareValorantCodeIsTheGamesDefaultCrosshair(t *testing.T) {
	s := read(t, "0")
	want := crosshair.Parts{
		Inner:          lines(true, 80, 6, 2, 3),
		Outer:          lines(true, 35, 2, 2, 10),
		Dot:            crosshair.Spot{On: false, Opacity: 100, Size: 2},
		Ring:           s.Parts.Ring,
		OutlineOpacity: 50,
	}

	if !s.Built || s.Color != (crosshair.RGB{R: 255, G: 255, B: 255}) || s.Outline.Width != 1 || s.Outline.Color != (crosshair.RGB{}) || s.Parts != want || s.Parts.Ring.On {
		t.Errorf("got %+v, want white with a one pixel outline and parts %+v", s, want)
	}
}

func TestAValorantCodeSetsOnlyWhatItNames(t *testing.T) {
	s := read(t, "0;P;c;5;h;0;0l;4;0o;2;0a;1;0f;0;1b;0")

	if s.Color != (crosshair.RGB{G: 255, B: 255}) || s.Outline.Width != 0 {
		t.Errorf("got colour %v and outline %d, want cyan without an outline", s.Color, s.Outline.Width)
	}
	if want := lines(true, 100, 4, 2, 2); s.Parts.Inner != want || s.Parts.Outer.On || s.Parts.Dot.On {
		t.Errorf("got inner lines %+v, outer %v and dot %v; want %+v alone", s.Parts.Inner, s.Parts.Outer.On, s.Parts.Dot.On, want)
	}
}

func TestADotCodeShowsTheDotAlone(t *testing.T) {
	s := read(t, "0;P;h;0;d;1;z;3;a;0.5;0b;0;1b;0")

	if want := (crosshair.Spot{On: true, Opacity: 50, Size: 3}); s.Parts.Dot != want || s.Parts.Inner.On || s.Parts.Outer.On {
		t.Errorf("got dot %+v with inner lines %v and outer %v, want %+v alone", s.Parts.Dot, s.Parts.Inner.On, s.Parts.Outer.On, want)
	}
}

func TestACustomColourLaysItsOpacityOverEveryPart(t *testing.T) {
	s := read(t, "0;P;c;8;u;00B5FF80")

	if s.Color != (crosshair.RGB{G: 0xB5, B: 0xFF}) || s.Parts.Inner.Opacity != 40 || s.Parts.Outer.Opacity != 18 || s.Parts.Dot.Opacity != 50 || s.Parts.OutlineOpacity != 50 {
		t.Errorf("got colour %v, opacities %d inner, %d outer, %d dot and %d outline", s.Color, s.Parts.Inner.Opacity, s.Parts.Outer.Opacity, s.Parts.Dot.Opacity, s.Parts.OutlineOpacity)
	}
	if s := read(t, "0;P;c;8;u;102030"); s.Color != (crosshair.RGB{R: 0x10, G: 0x20, B: 0x30}) || s.Parts.Inner.Opacity != 80 {
		t.Errorf("a colour without its opacity gives %v at %d", s.Color, s.Parts.Inner.Opacity)
	}
}

func TestOnlyThePrimaryCrosshairIsRead(t *testing.T) {
	s := read(t, "0;s;1;c;6;P;c;1;0l;5;A;c;7;0l;9;S;c;4;")

	if s.Color != (crosshair.RGB{G: 255}) || s.Parts.Inner.Length != 5 {
		t.Errorf("got colour %v and length %d, want the primary's green and 5", s.Color, s.Parts.Inner.Length)
	}
}

func TestALineOfNoLengthOrThicknessIsNotShown(t *testing.T) {
	if s := read(t, "0;P;0l;0;1t;0"); s.Parts.Inner.On || s.Parts.Outer.On {
		t.Errorf("inner lines of no length show %v and outer of no thickness %v", s.Parts.Inner.On, s.Parts.Outer.On)
	}
}

func TestValorantValuesPastTheRangeHereAreHeldToIt(t *testing.T) {
	s := read(t, "0;P;0l;99;0t;30;0o;90;t;6;z;20;0a;3")

	if want := lines(true, 100, 64, 16, 64); s.Parts.Inner != want || s.Outline.Width != 4 || s.Parts.Dot.Size != 16 {
		t.Errorf("got inner lines %+v, outline %d and dot %d", s.Parts.Inner, s.Outline.Width, s.Parts.Dot.Size)
	}
}

func TestADamagedValorantCodeIsRefused(t *testing.T) {
	for _, code := range []string{"0;P;c", "0;P;0l;abc", "0;P;c;9", "0;P;c;8;u;XYZ", "0;P;c;8", "0;P;;1", "0;P;0l;NaN"} {
		if _, err := gamecode.Read(code); !errors.Is(err, gamecode.ErrBroken) {
			t.Errorf("%q gives %v, want ErrBroken", code, err)
		}
	}
}

func TestWhatIsNoGameCodeIsRefused(t *testing.T) {
	for _, code := range []string{"", "hello", "1;P;c;5", "np1.cross-dot.00FF66.100.8.2.3.10.2.1.000000", "00;P"} {
		if _, err := gamecode.Read(code); !errors.Is(err, gamecode.ErrUnknown) {
			t.Errorf("%q gives %v, want ErrUnknown", code, err)
		}
	}
}

// The colours as players' guides list them for the numbers after c.
func TestEveryValorantColourNumberGivesItsColour(t *testing.T) {
	for number, hex := range []string{"#FFFFFF", "#00FF00", "#7FFF00", "#DFFF00", "#FFFF00", "#00FFFF", "#FF00FF", "#FF0000"} {
		var want crosshair.RGB
		if err := want.UnmarshalText([]byte(hex)); err != nil {
			t.Fatal(err)
		}
		if got := read(t, "0;P;c;"+string(rune('0'+number))).Color; got != want {
			t.Errorf("colour %d is %v, want %s", number, got, hex)
		}
	}
}
