package hotkeys_test

import (
	"errors"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

type bound struct {
	text      string
	modifiers uint32
	key       uint32
	repeat    bool
}

const (
	ctrlShift    = hotkeys.Control | hotkeys.Shift
	ctrlShiftAlt = hotkeys.Control | hotkeys.Shift | hotkeys.Alt
)

func TestTheSpecDefaultsBindElevenKeys(t *testing.T) {
	got, err := hotkeys.Bindings(profiles.Defaults().Hotkeys)
	if err != nil {
		t.Fatal(err)
	}

	want := []bound{
		{"Ctrl+Shift+X", ctrlShift, 'X', false},
		{"Ctrl+Shift+Left", ctrlShift, 0x25, true},
		{"Ctrl+Shift+Right", ctrlShift, 0x27, true},
		{"Ctrl+Shift+Up", ctrlShift, 0x26, true},
		{"Ctrl+Shift+Down", ctrlShift, 0x28, true},
		{"Ctrl+Shift+Alt+Left", ctrlShiftAlt, 0x25, true},
		{"Ctrl+Shift+Alt+Right", ctrlShiftAlt, 0x27, true},
		{"Ctrl+Shift+Alt+Up", ctrlShiftAlt, 0x26, true},
		{"Ctrl+Shift+Alt+Down", ctrlShiftAlt, 0x28, true},
		{"Ctrl+Shift+Home", ctrlShift, 0x24, false},
		{"Ctrl+Shift+C", ctrlShift, 'C', false},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d bindings, want %d", len(got), len(want))
	}
	for i, b := range got {
		if (bound{b.Text, b.Modifiers, b.Key, b.Repeat}) != want[i] {
			t.Errorf("binding %d is %+v, want %+v", i, b, want[i])
		}
	}
}

func TestOnlyTheArrowsRepeatWhileHeld(t *testing.T) {
	got, err := hotkeys.Bindings(profiles.Defaults().Hotkeys)
	if err != nil {
		t.Fatal(err)
	}

	for _, b := range got {
		want := b.Modifiers | hotkeys.NoRepeat
		if b.Repeat {
			want = b.Modifiers
		}
		if b.Flags() != want {
			t.Errorf("%s registers with %#x, want %#x", b.Text, b.Flags(), want)
		}
	}
	if got[0].Flags()&hotkeys.NoRepeat == 0 || got[1].Flags()&hotkeys.NoRepeat != 0 {
		t.Errorf("the toggle registers with %#x and a nudge with %#x; want the toggle alone not to repeat", got[0].Flags(), got[1].Flags())
	}
}

func TestEveryBindingCarriesItsOwnActionNumber(t *testing.T) {
	got, err := hotkeys.Bindings(profiles.Defaults().Hotkeys)
	if err != nil {
		t.Fatal(err)
	}

	seen := map[int]bool{}
	for _, b := range got {
		if seen[b.Action] {
			t.Errorf("action %d is carried twice", b.Action)
		}
		seen[b.Action] = true
	}
}

func TestAHotkeyIsReadInAnyCaseAndOrderAndWrittenOneWay(t *testing.T) {
	keys := profiles.Defaults().Hotkeys
	keys.Toggle = " shift + CONTROL + f9 "
	keys.Nudge = "alt+ctrl"

	got, err := hotkeys.Bindings(keys)
	if err != nil {
		t.Fatal(err)
	}

	if b := got[0]; b.Text != "Ctrl+Shift+F9" || b.Modifiers != ctrlShift || b.Key != 0x78 {
		t.Errorf("toggle read as %+v, want Ctrl+Shift+F9", b)
	}
	if b := got[1]; b.Text != "Ctrl+Alt+Left" || b.Modifiers != hotkeys.Control|hotkeys.Alt {
		t.Errorf("nudge left read as %+v, want Ctrl+Alt+Left", b)
	}
}

func TestKeysAreKnownByTheirNames(t *testing.T) {
	codes := map[string]uint32{
		"A": 0x41, "z": 0x5A, "0": 0x30, "9": 0x39,
		"F1": 0x70, "F12": 0x7B, "F24": 0x87,
		"Numpad0": 0x60, "Numpad9": 0x69,
		"Left": 0x25, "Up": 0x26, "Right": 0x27, "Down": 0x28,
		"Home": 0x24, "End": 0x23, "PageUp": 0x21, "PageDown": 0x22,
		"Insert": 0x2D, "Delete": 0x2E, "Space": 0x20,
	}

	for name, code := range codes {
		keys := profiles.Defaults().Hotkeys
		keys.Toggle = "Win+" + name

		got, err := hotkeys.Bindings(keys)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got[0].Key != code || got[0].Modifiers != hotkeys.Win {
			t.Errorf("%s: key %#x with modifiers %#x, want %#x with Win alone", name, got[0].Key, got[0].Modifiers, code)
		}
	}
}

func TestMalformedHotkeysAreRefused(t *testing.T) {
	cases := map[string]func(*profiles.Hotkeys){
		"a key alone":                   func(k *profiles.Hotkeys) { k.Toggle = "X" },
		"shift alone, which types":      func(k *profiles.Hotkeys) { k.Toggle = "Shift+X" },
		"no key":                        func(k *profiles.Hotkeys) { k.Toggle = "Ctrl+Shift" },
		"two keys":                      func(k *profiles.Hotkeys) { k.Toggle = "Ctrl+X+Y" },
		"a modifier twice":              func(k *profiles.Hotkeys) { k.Toggle = "Ctrl+Ctrl+X" },
		"an empty part":                 func(k *profiles.Hotkeys) { k.Toggle = "Ctrl++X" },
		"an unknown key":                func(k *profiles.Hotkeys) { k.Toggle = "Ctrl+F25" },
		"a function key below F1":       func(k *profiles.Hotkeys) { k.Toggle = "Ctrl+F0" },
		"a numpad key past 9":           func(k *profiles.Hotkeys) { k.Toggle = "Ctrl+Numpad10" },
		"nothing at all":                func(k *profiles.Hotkeys) { k.Center = "" },
		"a key where only modifiers go": func(k *profiles.Hotkeys) { k.Nudge = "Ctrl+Shift+X" },
		"no modifiers for the arrows":   func(k *profiles.Hotkeys) { k.NudgeFar = "" },
	}

	for name, change := range cases {
		keys := profiles.Defaults().Hotkeys
		change(&keys)

		if _, err := hotkeys.Bindings(keys); !errors.Is(err, hotkeys.ErrBadHotkey) {
			t.Errorf("%s: got %v, want ErrBadHotkey", name, err)
		}
	}
}

func TestTwoActionsCannotShareAHotkey(t *testing.T) {
	cases := map[string]func(*profiles.Hotkeys){
		"centre on an arrow":   func(k *profiles.Hotkeys) { k.Center = "Ctrl+Shift+Left" },
		"far nudges like near": func(k *profiles.Hotkeys) { k.NudgeFar = "Shift+Ctrl" },
		"next where toggle is": func(k *profiles.Hotkeys) { k.Next = "ctrl+shift+x" },
	}

	for name, change := range cases {
		keys := profiles.Defaults().Hotkeys
		change(&keys)

		if _, err := hotkeys.Bindings(keys); !errors.Is(err, hotkeys.ErrSameHotkey) {
			t.Errorf("%s: got %v, want ErrSameHotkey", name, err)
		}
	}
}
