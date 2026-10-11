package tray_test

import (
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/tray"
)

func TestWindowsStartsTheProgramByItsWholePath(t *testing.T) {
	if got, want := tray.StartCommand(`C:\Program Files\NormPrichel\normprichel.exe`), `"C:\Program Files\NormPrichel\normprichel.exe"`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestTaskManagerSwitchesAProgramOffByAnOddFirstByte(t *testing.T) {
	cases := []struct {
		mark []byte
		want bool
	}{
		{mark: nil, want: true},
		{mark: []byte{0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, want: true},
		{mark: []byte{0x06, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, want: true},
		{mark: []byte{0x03, 0x9C, 0x5F, 0x2B, 0x41, 0x8E, 0xD9, 0x01, 0, 0, 0, 0}, want: false},
		{mark: []byte{0x07}, want: false},
	}

	for _, c := range cases {
		if got := tray.Approved(c.mark); got != c.want {
			t.Errorf("mark % x: got %v, want %v", c.mark, got, c.want)
		}
	}
}
