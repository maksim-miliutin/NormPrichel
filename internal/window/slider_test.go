package window_test

import (
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/window"
)

func everyKnob() []crosshair.Knob {
	return append(crosshair.Knobs(crosshair.CrossDot), crosshair.Knobs(crosshair.Circle)[2], crosshair.PictureScale())
}

func TestTheEndsOfTheTrackHoldTheEndsOfTheRange(t *testing.T) {
	track := window.Track{Width: 150, Thumb: 10}
	for _, knob := range everyKnob() {
		if low, high := track.ValueAt(knob, -40), track.ValueAt(knob, 400); low != knob.Min || high != knob.Max {
			t.Errorf("%s: the ends read %d and %d, want %d and %d", knob.Name, low, high, knob.Min, knob.Max)
		}
		if left, right := track.ThumbAt(knob, knob.Min), track.ThumbAt(knob, knob.Max); left != 0 || right != 140 {
			t.Errorf("%s: the thumb runs from %d to %d, want 0 to 140", knob.Name, left, right)
		}
	}
}

func TestAPressOnTheThumbReadsTheValueItShows(t *testing.T) {
	for _, track := range []window.Track{{Width: 150, Thumb: 10}, {Width: 225, Thumb: 15}, {Width: 112, Thumb: 12}} {
		for _, knob := range everyKnob() {
			for value := knob.Min; value <= knob.Max; value += knob.Step {
				at := track.ThumbAt(knob, value) + track.Thumb/2
				if got := track.ValueAt(knob, at); got != value {
					t.Errorf("%s on a %d pixel track: the thumb of %d at %d reads %d", knob.Name, track.Width, value, at, got)
				}
			}
		}
	}
}

func TestTheValueKeepsToTheStepsOfItsKnob(t *testing.T) {
	track, scale := window.Track{Width: 150, Thumb: 10}, crosshair.PictureScale()
	for x := range 150 {
		if got := track.ValueAt(scale, x); (got-scale.Min)%scale.Step != 0 {
			t.Fatalf("at %d the scale reads %d, off its steps of %d", x, got, scale.Step)
		}
	}
}

func TestKeysMoveByAStepOrAPageAndStopAtTheEnds(t *testing.T) {
	scale := crosshair.PictureScale()
	for _, c := range []struct {
		key   uint32
		from  int
		want  int
		taken bool
	}{
		{key: 0x27, from: 100, want: 125, taken: true},
		{key: 0x26, from: 100, want: 125, taken: true},
		{key: 0x25, from: 100, want: 75, taken: true},
		{key: 0x28, from: 100, want: 75, taken: true},
		{key: 0x21, from: 100, want: 350, taken: true},
		{key: 0x22, from: 300, want: 50, taken: true},
		{key: 0x22, from: 100, want: 25, taken: true},
		{key: 0x27, from: 400, want: 400, taken: true},
		{key: 0x24, from: 300, want: 25, taken: true},
		{key: 0x23, from: 50, want: 400, taken: true},
		{key: 'A', from: 100, want: 100, taken: false},
	} {
		if got, taken := window.Keyed(scale, c.from, c.key); got != c.want || taken != c.taken {
			t.Errorf("key %#x from %d: %d, %v; want %d, %v", c.key, c.from, got, taken, c.want, c.taken)
		}
	}
}

func TestAKnobWithNoRangeOrNoStepKeepsItsThumbAtTheStart(t *testing.T) {
	track := window.Track{Width: 150, Thumb: 10}
	for _, knob := range []crosshair.Knob{{Name: "still", Min: 5, Max: 5, Step: 1}, {Name: "stepless", Min: 5, Max: 10}} {
		if at, value := track.ThumbAt(knob, 7), track.ValueAt(knob, 90); at != 0 || value != 5 {
			t.Errorf("%s: thumb at %d reading %d, want 0 and 5", knob.Name, at, value)
		}
	}
}

func TestAValuePastTheRangeKeepsTheThumbOnTheTrack(t *testing.T) {
	track, scale := window.Track{Width: 150, Thumb: 10}, crosshair.PictureScale()

	if low, high := track.ThumbAt(scale, -100), track.ThumbAt(scale, 900); low != 0 || high != 140 {
		t.Errorf("the thumb sits at %d and %d, want 0 and 140", low, high)
	}
}
