package hotkeys_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

func TestTheFieldsComeInTheOrderOfTheSettingsFile(t *testing.T) {
	if got := hotkeys.Fields(); !slices.Equal(got, []string{"toggle", "nudge", "nudgeFar", "center", "next"}) {
		t.Errorf("got %v", got)
	}
}

func TestAFieldIsReadAndWrittenByName(t *testing.T) {
	keys := profiles.Defaults().Hotkeys

	if got := hotkeys.Get(keys, "next"); got != "Ctrl+Shift+C" {
		t.Errorf("next reads %q", got)
	}
	changed, err := hotkeys.Set(keys, "next", "Ctrl+Alt+N")
	if err != nil || hotkeys.Get(changed, "next") != "Ctrl+Alt+N" || hotkeys.Get(keys, "next") != "Ctrl+Shift+C" {
		t.Errorf("got %+v, %v; want next changed in a copy", changed, err)
	}
	if _, err := hotkeys.Set(keys, "jump", "Ctrl+J"); !errors.Is(err, hotkeys.ErrNoField) {
		t.Errorf("got %v, want ErrNoField", err)
	}
	if got := hotkeys.Get(keys, "jump"); got != "" {
		t.Errorf("an unknown field reads %q", got)
	}
}

func TestOnlyTheNudgeFieldsHoldModifiersForTheArrows(t *testing.T) {
	for _, field := range hotkeys.Fields() {
		if got, want := hotkeys.Arrows(field), field == "nudge" || field == "nudgeFar"; got != want {
			t.Errorf("%s: arrows %v, want %v", field, got, want)
		}
	}
}

func TestAPressIsTakenOnlyWithCtrlAltOrWinHeld(t *testing.T) {
	cases := []struct {
		modifiers, key uint32
		want           string
		ok             bool
	}{
		{modifiers: hotkeys.Control | hotkeys.Shift, key: 'X', want: "Ctrl+Shift+X", ok: true},
		{modifiers: hotkeys.Alt, key: 0x70, want: "Alt+F1", ok: true},
		{modifiers: hotkeys.Win | hotkeys.Shift, key: 0x67, want: "Shift+Win+Numpad7", ok: true},
		{modifiers: hotkeys.Control, key: 0x24, want: "Ctrl+Home", ok: true},
		{modifiers: hotkeys.Control, key: '5', want: "Ctrl+5", ok: true},
		{modifiers: hotkeys.Shift, key: 'X'},
		{modifiers: 0, key: 'X'},
		{modifiers: hotkeys.Control, key: 0x11},
		{modifiers: hotkeys.Control | hotkeys.Alt, key: 0x12},
	}

	for _, c := range cases {
		if got, ok := hotkeys.Pressed("toggle", c.modifiers, c.key); ok != c.ok || got != c.want {
			t.Errorf("modifiers %#x with key %#x: got %q, %v; want %q, %v", c.modifiers, c.key, got, ok, c.want, c.ok)
		}
	}
}

func TestAnArrowFieldTakesTheModifiersOfAnArrowPress(t *testing.T) {
	if got, ok := hotkeys.Pressed("nudge", hotkeys.Control|hotkeys.Alt, 0x26); !ok || got != "Ctrl+Alt" {
		t.Errorf("Ctrl+Alt+Up gives %q, %v; want Ctrl+Alt", got, ok)
	}
	if got, ok := hotkeys.Pressed("nudgeFar", hotkeys.Control|hotkeys.Shift, 'X'); ok {
		t.Errorf("a letter in an arrow field gives %q", got)
	}
}
