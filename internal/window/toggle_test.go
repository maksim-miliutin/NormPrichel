package window_test

import (
	"image"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/window"
)

func switchOf(on bool) window.Toggle {
	return window.Toggle{Size: image.Pt(200, 24), DPI: 96, On: on}
}

func TestTheKnobSitsAtTheLeftWhenOffAndAtTheRightWhenOn(t *testing.T) {
	off, on := switchOf(false), switchOf(true)
	track, left, right := off.Track(), off.Knob(), on.Knob()

	if !left.In(track) || !right.In(track) {
		t.Fatalf("the knob %v / %v leaves the track %v", left, right, track)
	}
	if left.Dx() != left.Dy() || right.Size() != left.Size() || right.Min.Y != left.Min.Y {
		t.Errorf("the knob goes from %v to %v, want the same round knob slid along", left, right)
	}
	if middle := track.Min.X + track.Dx()/2; left.Max.X > middle || right.Min.X < middle {
		t.Errorf("the knob sits at %v when off and %v when on, want the left and the right half of %v", left, right, track)
	}
	if fromLeft, fromRight := left.Min.X-track.Min.X, track.Max.X-right.Max.X; fromLeft != fromRight || fromLeft <= 0 {
		t.Errorf("the knob keeps %d pixels from the left end and %d from the right, want the same, above zero", fromLeft, fromRight)
	}
	if top, bottom := left.Min.Y-track.Min.Y, track.Max.Y-left.Max.Y; top != bottom {
		t.Errorf("the knob sits %d pixels from the top and %d from the bottom, want it centred", top, bottom)
	}
}

func TestAPressedKnobStretchesTowardTheMiddle(t *testing.T) {
	for _, on := range []bool{false, true} {
		still := switchOf(on)
		pressed := still
		pressed.Pressed = true
		before, after := still.Knob(), pressed.Knob()

		kept := after.Min.X == before.Min.X
		if on {
			kept = after.Max.X == before.Max.X
		}
		if after.Dx() <= before.Dx() || after.Dy() != before.Dy() || !kept {
			t.Errorf("on %v: pressing turns the knob %v into %v, want it wider toward the middle", on, before, after)
		}
	}
}

func TestASwitchTrackSitsAtTheStartOfItsControlAndGrowsWithTheScale(t *testing.T) {
	control := image.Pt(390, 40)
	for dpi, want := range map[int]image.Point{96: {36, 18}, 144: {54, 27}} {
		track := window.Toggle{Size: control, DPI: dpi}.Track()
		if track.Size() != want || track.Min.X != 0 {
			t.Errorf("at %d dpi the track is %v, want %v at the left end", dpi, track, want)
		}
		if top, bottom := track.Min.Y, control.Y-track.Max.Y; top-bottom > 1 || bottom-top > 1 {
			t.Errorf("at %d dpi the track sits %d from the top and %d from the bottom, want it centred", dpi, top, bottom)
		}
	}
}

func TestASwitchedOnTrackGlowsAmberAndAnOffOneShowsItsOutline(t *testing.T) {
	off, on := switchOf(false), switchOf(true)
	track := off.Track()
	middle := track.Min.Y + track.Dy()/2
	centre := func(s window.Toggle) image.Point {
		knob := s.Knob()
		return image.Pt(knob.Min.X+knob.Dx()/2, knob.Min.Y+knob.Dy()/2)
	}
	offKnob, onKnob := centre(off), centre(on)

	glow, onDot := at(on.Image(), track.Min.X+4, middle), at(on.Image(), onKnob.X, onKnob.Y)
	if !amber(glow) || brightness(onDot) >= brightness(glow) {
		t.Errorf("a switch that is on shows %v with a knob of %v, want amber with a dark knob", glow, onDot)
	}
	inside, outline, offDot := at(off.Image(), track.Max.X-5, middle), at(off.Image(), track.Max.X-1, middle), at(off.Image(), offKnob.X, offKnob.Y)
	if amber(inside) || brightness(outline) <= brightness(inside) || brightness(offDot) <= brightness(inside) {
		t.Errorf("a switch that is off shows %v inside an outline of %v with a knob of %v, want a quiet track, a lighter outline and knob", inside, outline, offDot)
	}
}

func TestASwitchWithTheKeysWearsALightEdge(t *testing.T) {
	for _, on := range []bool{false, true} {
		plain := switchOf(on)
		keyed := plain
		keyed.Keyed = true
		track := plain.Track()

		for x := track.Min.X; x < track.Min.X+2; x++ {
			before, after := at(plain.Image(), x, track.Min.Y+track.Dy()/2), at(keyed.Image(), x, track.Min.Y+track.Dy()/2)
			if brightness(after) <= brightness(before) || amber(after) {
				t.Errorf("on %v: column %d of the edge is %v with the keys and %v without, want a light edge", on, x, after, before)
			}
		}
	}
}
