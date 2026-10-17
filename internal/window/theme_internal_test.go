package window

import (
	"math"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

// WCAG 2 contrast: the relative luminance of the lighter colour plus 0.05 over the darker one's.
func contrast(a, b crosshair.RGB) float64 {
	luminance := func(c crosshair.RGB) float64 {
		linear := func(v uint8) float64 {
			s := float64(v) / 255
			if s <= 0.04045 {
				return s / 12.92
			}
			return math.Pow((s+0.055)/1.055, 2.4)
		}
		return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
	}
	light, dark := luminance(a), luminance(b)
	if light < dark {
		light, dark = dark, light
	}

	return (light + 0.05) / (dark + 0.05)
}

func TestEveryWordReadsOnWhatItIsWrittenOn(t *testing.T) {
	for _, c := range []struct {
		name         string
		words, paper crosshair.RGB
	}{
		{"text on the window", ink, ground},
		{"text in a field", ink, field},
		{"a hint on the window", ash, ground},
		{"words on what is on", onSignal, signal},
		{"words on a refusal", ink, rust},
	} {
		if got := contrast(c.words, c.paper); got < 4.5 {
			t.Errorf("%s: contrast %.2f, want at least 4.5", c.name, got)
		}
	}
}

func TestWhatIsOnStandsOutAndFieldsLiftOffTheWindow(t *testing.T) {
	if got := contrast(signal, field); got < 3 {
		t.Errorf("a chosen control against an idle one: contrast %.2f, want at least 3", got)
	}
	if got := contrast(field, ground); got < 1.1 {
		t.Errorf("a field against the window: contrast %.2f, want it a shade apart", got)
	}
	if got := contrast(seam, field); got < 1.3 {
		t.Errorf("an edge against its field: contrast %.2f, want it visible", got)
	}
}
