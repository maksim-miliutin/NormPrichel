package crosshair_test

import (
	"errors"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

// The site parses the same code; this one is the example both sides are held to.
const defaultCode = "np1.cross-dot.00FF66.100.8.2.3.10.2.1.000000"

func TestTheDefaultCrosshairReadsAsItsCode(t *testing.T) {
	if got := crosshair.Default().Code(); got != defaultCode {
		t.Errorf("the default crosshair has the code %q, want %q", got, defaultCode)
	}
}

func TestACodeWritesTheValuesTheDrawingKeepsTo(t *testing.T) {
	wild := crosshair.Style{Shape: crosshair.Cross, Opacity: 0, Length: 99, Thickness: 0, Gap: 40, Radius: 1, DotSize: 0, Outline: crosshair.Outline{Width: 9}}

	if got, want := wild.Code(), "np1.cross.000000.10.64.1.32.2.1.4.000000"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEveryShapeComesBackFromItsCode(t *testing.T) {
	for i, shape := range crosshair.Shapes() {
		style := crosshair.Style{
			Shape: shape, Color: crosshair.RGB{R: uint8(i), G: 0xA0, B: 0x0F}, Opacity: 40 + i,
			Length: 1 + i, Thickness: 1 + i%16, Gap: i, Radius: 2 + i, DotSize: 1 + i%16,
			Outline: crosshair.Outline{Width: i % 5, Color: crosshair.RGB{R: 0xFE, G: uint8(i)}},
		}

		got, err := crosshair.ParseCode(style.Code())
		if err != nil || got != style {
			t.Errorf("%s: the code %q gives back %+v, %v; want %+v", shape, style.Code(), got, err, style)
		}
	}
}

func TestACodeIsReadInEitherCaseWithSpaceAround(t *testing.T) {
	got, err := crosshair.ParseCode(" NP1.cross-dot.00ff66.100.8.2.3.10.2.1.000000\r\n")
	if err != nil || got != crosshair.Default() {
		t.Errorf("got %+v, %v; want the default crosshair", got, err)
	}
}

func TestValuesPastTheirRangeInACodeAreHeldToTheLimits(t *testing.T) {
	got, err := crosshair.ParseCode("np1.cross.00FF66.5.99.0.40.1.0.9.000000")
	want := crosshair.Style{Shape: crosshair.Cross, Color: crosshair.RGB{G: 0xFF, B: 0x66}, Opacity: 10, Length: 64, Thickness: 1, Gap: 32, Radius: 2, DotSize: 1, Outline: crosshair.Outline{Width: 4}}
	if err != nil || got != want {
		t.Errorf("got %+v, %v; want %+v", got, err, want)
	}
}

func TestWhatIsNotACrosshairCodeIsRefused(t *testing.T) {
	for _, text := range []string{
		"",
		"hello",
		"np2.cross-dot.00FF66.100.8.2.3.10.2.1.000000",
		"np1.cross-dot.00FF66.100.8.2.3.10.2.1",
		"np1.cross-dot.00FF66.100.8.2.3.10.2.1.000000.7",
		"np1.star.00FF66.100.8.2.3.10.2.1.000000",
		"np1.cross-dot.#0FF66.100.8.2.3.10.2.1.000000",
		"np1.cross-dot.00FF6.100.8.2.3.10.2.1.000000",
		"np1.cross-dot.00FF66.100.8.-2.3.10.2.1.000000",
		"np1.cross-dot.00FF66.100.8.2.3.10.2.1.00000G",
		"np1.cross-dot.00FF66.1000.8.2.3.10.2.1.000000",
		"np1.cross-dot.00FF66.100..2.3.10.2.1.000000",
		"np1.cross-dot.00FF66.100.8.2.3.10.2.1.000000.round.shadow",
		"np2.cross-dot.00FF66.100.8.2.3.10.2.1.000000.round",
		"np2.cross-dot.00FF66.100.8.2.3.10.2.1.000000.squircle.line",
		"np2.cross-dot.00FF66.100.8.2.3.10.2.1.000000.round.glow",
	} {
		if _, err := crosshair.ParseCode(text); !errors.Is(err, crosshair.ErrBadCode) {
			t.Errorf("%q gives %v, want ErrBadCode", text, err)
		}
	}
}

func TestRoundingAndAShadowGoIntoASecondVersionOfTheCode(t *testing.T) {
	style := crosshair.Default()
	style.Corners, style.Outline.Shadow = crosshair.Round, true
	if got, want := style.Code(), "np2.cross-dot.00FF66.100.8.2.3.10.2.1.000000.round.shadow"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	style.Corners, style.Outline.Shadow = crosshair.Soft, false
	if got, want := style.Code(), "np2.cross-dot.00FF66.100.8.2.3.10.2.1.000000.soft.line"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEveryCornerAndOutlineComesBackFromItsCode(t *testing.T) {
	for _, corners := range []crosshair.Corners{crosshair.Sharp, crosshair.Soft, crosshair.Round} {
		for _, shadow := range []bool{false, true} {
			style := crosshair.Default()
			style.Corners, style.Outline.Shadow = corners, shadow

			got, err := crosshair.ParseCode(style.Code())
			if err != nil || got != style {
				t.Errorf("the code %q gives back %+v, %v; want %+v", style.Code(), got, err, style)
			}
		}
	}

	got, err := crosshair.ParseCode("NP2.cross-dot.00ff66.100.8.2.3.10.2.1.000000.SHARP.Line")
	if err != nil || got != crosshair.Default() {
		t.Errorf("a second version code of a plain crosshair gives %+v, %v; want the default", got, err)
	}
}

func TestAShareOfRoundingGoesIntoTheCornersField(t *testing.T) {
	style := crosshair.Default()
	style.Rounding = 35
	code := style.Code()
	if want := "np2.cross-dot.00FF66.100.8.2.3.10.2.1.000000.35.line"; code != want {
		t.Errorf("got %q, want %q", code, want)
	}

	got, err := crosshair.ParseCode(code)
	if err != nil || got != style {
		t.Errorf("the code %q gives back %+v, %v; want %+v", code, got, err, style)
	}
}

// Older copies of the program read the words, so the ends of the way are written as words.
func TestTheEndsOfTheWayToRoundAreWrittenAsTheirWords(t *testing.T) {
	for _, c := range []struct {
		rounding int
		want     string
	}{
		{100, "np2.cross-dot.00FF66.100.8.2.3.10.2.1.000000.round.line"},
		{150, "np2.cross-dot.00FF66.100.8.2.3.10.2.1.000000.round.line"},
		{0, defaultCode},
	} {
		style := crosshair.Default()
		style.Rounding = c.rounding
		if got := style.Code(); got != c.want {
			t.Errorf("rounded %d%%: got %q, want %q", c.rounding, got, c.want)
		}
	}
}

func TestASharePastTheEndsOfTheWayIsReadAsTheEnd(t *testing.T) {
	round := crosshair.Default()
	round.Corners = crosshair.Round
	for field, want := range map[string]crosshair.Style{"0": crosshair.Default(), "000": crosshair.Default(), "100": round, "250": round} {
		got, err := crosshair.ParseCode("np2.cross-dot.00FF66.100.8.2.3.10.2.1.000000." + field + ".line")
		if err != nil || got != want {
			t.Errorf("%q: got %+v, %v; want %+v", field, got, err, want)
		}
	}
}

func TestAShareThatIsNotAWholeNumberIsRefused(t *testing.T) {
	for _, field := range []string{"-5", "5x", "1234", "", "4,5", "+7"} {
		code := "np2.cross-dot.00FF66.100.8.2.3.10.2.1.000000." + field + ".line"
		if _, err := crosshair.ParseCode(code); !errors.Is(err, crosshair.ErrBadCode) {
			t.Errorf("%q gives %v, want ErrBadCode", code, err)
		}
	}
}
