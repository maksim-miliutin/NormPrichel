package mouse

import "testing"

func TestOnlyTheTopBitOfTheStateMeansTheButtonIsHeld(t *testing.T) {
	for _, c := range []struct {
		state uintptr
		held  bool
		why   string
	}{
		{0x0000, false, "nothing set"},
		{0x8000, true, "the top bit"},
		{0x8001, true, "the top bit with a press since the last look"},
		{0x0001, false, "a press since the last look, let go since"},
		{0xFFFF8000, true, "the top bit spread over the upper half"},
		{0xFFFF0000, false, "noise above the sixteen bits"},
	} {
		if got := held(c.state); got != c.held {
			t.Errorf("%#x (%s) reads as held %v, want %v", c.state, c.why, got, c.held)
		}
	}
}
