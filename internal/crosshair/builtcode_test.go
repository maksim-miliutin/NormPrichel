package crosshair_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

const builtDefaultCode = "np3.00FF66.1.000000.100.sharp.line.1.100.8.2.3.1111.0.100.4.2.15.1111.1.100.2.0.100.10.2"

func TestTheDefaultCrosshairBuiltReadsAsItsCode(t *testing.T) {
	if got := crosshair.Default().Build().Code(); got != builtDefaultCode {
		t.Errorf("got %q, want %q", got, builtDefaultCode)
	}

	got, err := crosshair.ParseCode(builtDefaultCode)
	if want := crosshair.Default().Build(); err != nil || got != want {
		t.Errorf("got %+v, %v; want %+v", got, err, want)
	}
}

// Every value differs from the others, so a field read into another's place shows.
func TestEveryBuiltFieldComesBackFromItsCode(t *testing.T) {
	style := crosshair.Default()
	style.Built, style.Rounding = true, 35
	style.Color, style.Outline = crosshair.RGB{R: 1, G: 2, B: 3}, crosshair.Outline{Width: 3, Color: crosshair.RGB{R: 4, G: 5, B: 6}, Shadow: true}
	style.Parts = crosshair.Parts{
		Inner:          crosshair.Lines{On: true, Opacity: 11, Length: 12, Thickness: 13, Gap: 14, Hidden: crosshair.Arms{Up: true}},
		Outer:          crosshair.Lines{On: false, Opacity: 21, Length: 22, Thickness: 9, Gap: 24, Hidden: crosshair.Arms{Right: true, Left: true}},
		Dot:            crosshair.Spot{On: true, Opacity: 31, Size: 7},
		Ring:           crosshair.Band{On: true, Opacity: 41, Radius: 42, Thickness: 5},
		OutlineOpacity: 51,
	}
	code := style.Code()

	got, err := crosshair.ParseCode(code)
	if err != nil || got != style {
		t.Errorf("the code %q gives back %+v, %v; want %+v", code, got, err, style)
	}
	if want := "np3.010203.3.040506.51.35.shadow.1.11.12.13.14.0111.0.21.22.9.24.1010.1.31.7.1.41.42.5"; code != want {
		t.Errorf("got %q, want %q", code, want)
	}
}

func TestABuiltCodeIsReadInEitherCaseWithSpaceAround(t *testing.T) {
	got, err := crosshair.ParseCode(" NP3.00ff66.1.000000.100.SHARP.Line.1.100.8.2.3.1111.0.100.4.2.15.1111.1.100.2.0.100.10.2\r\n")
	if want := crosshair.Default().Build(); err != nil || got != want {
		t.Errorf("got %+v, %v; want the default crosshair built", got, err)
	}
}

func TestValuesPastTheirRangeInABuiltCodeAreHeldToTheLimits(t *testing.T) {
	got, err := crosshair.ParseCode("np3.00FF66.9.000000.250.100.line.1.100.99.0.80.1111.0.100.4.2.15.1111.1.100.2.0.100.10.2")
	if err != nil {
		t.Fatal(err)
	}
	inner := got.Parts.Inner
	if got.Outline.Width != 4 || got.Parts.OutlineOpacity != 100 || got.Corners != crosshair.Round || inner.Length != 64 || inner.Thickness != 1 || inner.Gap != 64 {
		t.Errorf("got %+v", got)
	}
}

func TestWhatIsNotABuiltCodeIsRefused(t *testing.T) {
	good := strings.Split(builtDefaultCode, ".")
	broken := func(at int, field string) string {
		fields := append([]string{}, good...)
		fields[at] = field
		return strings.Join(fields, ".")
	}
	for _, text := range []string{
		strings.Join(good[:len(good)-1], "."),
		builtDefaultCode + ".2",
		"np2" + strings.TrimPrefix(builtDefaultCode, "np3"),
		broken(1, "GG0000"),
		broken(2, "-1"),
		broken(4, "1000"),
		broken(5, "squircle"),
		broken(6, "glow"),
		broken(7, "2"),
		broken(12, "111"),
		broken(12, "11a1"),
		broken(19, "yes"),
	} {
		if _, err := crosshair.ParseCode(text); !errors.Is(err, crosshair.ErrBadCode) {
			t.Errorf("%q gives %v, want ErrBadCode", text, err)
		}
	}
}
