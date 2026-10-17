package window_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/settings"
	"github.com/maksim-miliutin/NormPrichel/internal/window"
)

func TestEveryControlLiesInsideTheWindow(t *testing.T) {
	w, h := window.Size()
	for _, p := range window.Places() {
		if p.Rect.X < 0 || p.Rect.Y < 0 || p.Rect.W <= 0 || p.Rect.H <= 0 || p.Rect.X+p.Rect.W > w || p.Rect.Y+p.Rect.H > h {
			t.Errorf("control %d at %+v leaves the %d by %d window", p.ID, p.Rect, w, h)
		}
	}
}

func TestEveryControlHasItsOwnNumberClearOfTheDialogKeys(t *testing.T) {
	seen := map[int]bool{}
	for _, p := range window.Places() {
		if seen[p.ID] || p.ID < 100 {
			t.Errorf("control number %d is taken twice or clashes with Enter and Esc", p.ID)
		}
		seen[p.ID] = true
	}
}

func TestEdgesScaleSoNeighboursStayNeighbours(t *testing.T) {
	a, b := window.Rect{X: 13, Y: 12, W: 13, H: 26}, window.Rect{X: 26, Y: 12, W: 13, H: 26}
	for _, dpi := range []int{96, 120, 144, 168, 192, 240} {
		sa, sb := window.Scale(a, dpi), window.Scale(b, dpi)
		if sa.X+sa.W != sb.X {
			t.Errorf("at %d dpi %+v and %+v no longer touch", dpi, sa, sb)
		}
	}

	w, h := window.Size()
	if got := window.Scale(window.Rect{W: w, H: h}, 144); got != (window.Rect{W: w * 3 / 2, H: h * 3 / 2}) {
		t.Errorf("at 150%% the window is %+v", got)
	}
}

// A 1280 by 720 screen keeps 672 pixels under a 48 pixel taskbar, and the frame and title take
// another 16 across and 39 down.
func TestTheWindowFitsA1280By720Screen(t *testing.T) {
	if w, h := window.Size(); w+16 > 1280 || h+39 > 672 {
		t.Errorf("the window needs %d by %d", w+16, h+39)
	}
}

func TestEveryFixedWordIsThereInBothLanguages(t *testing.T) {
	for _, language := range []string{"ru", "en"} {
		for _, p := range window.Places() {
			if p.Word != "" && strings.TrimSpace(settings.Word(language, p.Word)) == "" {
				t.Errorf("%s: control %d shows %q, which has no words", language, p.ID, p.Word)
			}
		}
	}
}

func TestTheKnobsSitBetweenTheColourAndTheOutlineColour(t *testing.T) {
	var colour, outline window.Rect
	var sliders []window.Rect
	for _, p := range window.Places() {
		switch {
		case p.Word == "color":
			colour = p.Rect
		case p.Word == "outlineColor":
			outline = p.Rect
		case p.Kind == window.Slider && p.Group == "ready":
			sliders = append(sliders, p.Rect)
		}
	}
	slices.SortFunc(sliders, func(a, b window.Rect) int { return a.Y - b.Y })

	if len(sliders) != 6 {
		t.Fatalf("got %d slots for the knobs, want one for each of the six a shape may have", len(sliders))
	}
	if gap := sliders[0].Y - (colour.Y + colour.H); gap < 0 || gap > 12 {
		t.Errorf("the first knob starts %d pixels below the colour", gap)
	}
	if gap := outline.Y - (sliders[5].Y + sliders[5].H); gap < 0 || gap > 12 {
		t.Errorf("the outline colour starts %d pixels below the last knob", gap)
	}
}

func TestAFieldsOwnWindowSitsInsideItsFrameWithRoomForALine(t *testing.T) {
	for _, p := range window.Places() {
		inner, outer := window.Interior(p), p.Rect
		switch p.Kind {
		case window.Edit, window.Hotkey, window.List:
			inside := inner.X > outer.X && inner.Y > outer.Y && inner.X+inner.W < outer.X+outer.W && inner.Y+inner.H < outer.Y+outer.H
			if !inside || inner.H < 16 {
				t.Errorf("control %d: %+v within its frame %+v, want it inside with 16 pixels of height", p.ID, inner, outer)
			}
		default:
			if inner != outer {
				t.Errorf("control %d of kind %s takes %+v, want its whole place %+v", p.ID, p.Kind, inner, outer)
			}
		}
	}
}
