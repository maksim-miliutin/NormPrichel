package hotkeys_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

// heldElsewhere stands in for other programs: it reports as taken every binding whose text it
// was given, and counts how often it was asked.
type heldElsewhere struct {
	texts []string
	asked int
}

func (h *heldElsewhere) taken(bindings []hotkeys.Binding) []hotkeys.Binding {
	h.asked++

	var taken []hotkeys.Binding
	for _, b := range bindings {
		if slices.Contains(h.texts, b.Text) {
			taken = append(taken, b)
		}
	}

	return taken
}

func TestEveryBindingNamesTheSettingsFieldItComesFrom(t *testing.T) {
	keys := profiles.Defaults().Hotkeys
	text, err := json.Marshal(keys)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]string
	if err := json.Unmarshal(text, &fields); err != nil {
		t.Fatal(err)
	}

	bindings, err := hotkeys.Bindings(keys)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range bindings {
		value, ok := fields[b.Field]
		arrow, _ := strings.CutPrefix(b.Text, value+"+")
		if !ok || b.Text != value && !slices.Contains([]string{"Left", "Right", "Up", "Down"}, arrow) {
			t.Errorf("%s names field %q, which holds %q", b.Text, b.Field, value)
		}
	}
}

func TestATakenHotkeyIsOfferedTheSameKeyUnderOtherModifiers(t *testing.T) {
	elsewhere := &heldElsewhere{texts: []string{"Ctrl+Shift+C"}}

	got, ok := hotkeys.Suggest(profiles.Defaults().Hotkeys, "next", elsewhere.taken)
	if !ok || got != "Ctrl+Shift+Alt+C" {
		t.Errorf("got %q, %v; want Ctrl+Shift+Alt+C", got, ok)
	}
}

func TestTheMostFamiliarFreeModifiersComeFirst(t *testing.T) {
	keys := profiles.Defaults().Hotkeys
	keys.Toggle = "Ctrl+Alt+F9"
	elsewhere := &heldElsewhere{texts: []string{"Ctrl+Alt+F9"}}

	got, ok := hotkeys.Suggest(keys, "toggle", elsewhere.taken)
	if !ok || got != "Ctrl+Shift+F9" {
		t.Errorf("got %q, %v; want Ctrl+Shift+F9", got, ok)
	}
}

func TestASuggestionIsFreeItself(t *testing.T) {
	elsewhere := &heldElsewhere{texts: []string{"Ctrl+Shift+C", "Ctrl+Shift+Alt+C"}}

	got, ok := hotkeys.Suggest(profiles.Defaults().Hotkeys, "next", elsewhere.taken)
	if !ok || got != "Ctrl+Alt+C" {
		t.Errorf("got %q, %v; want Ctrl+Alt+C", got, ok)
	}
}

func TestASuggestionNeverLandsOnAnotherAction(t *testing.T) {
	keys := profiles.Defaults().Hotkeys
	keys.Toggle = "Ctrl+Shift+Alt+C"
	elsewhere := &heldElsewhere{texts: []string{"Ctrl+Shift+C"}}

	got, ok := hotkeys.Suggest(keys, "next", elsewhere.taken)
	if !ok || got != "Ctrl+Alt+C" {
		t.Errorf("got %q, %v; want Ctrl+Alt+C, past the toggle", got, ok)
	}
}

func TestASuggestionLooksOnlyAtItsOwnField(t *testing.T) {
	elsewhere := &heldElsewhere{texts: []string{"Ctrl+Shift+X", "Ctrl+Shift+C"}}

	got, ok := hotkeys.Suggest(profiles.Defaults().Hotkeys, "next", elsewhere.taken)
	if !ok || got != "Ctrl+Shift+Alt+C" {
		t.Errorf("got %q, %v; want Ctrl+Shift+Alt+C whatever holds the toggle", got, ok)
	}
}

func TestAHotkeyThatDoesNotReadGetsNoSuggestion(t *testing.T) {
	keys := profiles.Defaults().Hotkeys
	keys.Next = "Ctrl+F99"
	elsewhere := &heldElsewhere{}

	if got, ok := hotkeys.Suggest(keys, "next", elsewhere.taken); ok {
		t.Errorf("got %q, want no suggestion for a key that does not exist", got)
	}
}

func TestArrowsAreNeverOfferedCtrlAltWhichTurnsTheScreen(t *testing.T) {
	elsewhere := &heldElsewhere{texts: []string{"Ctrl+Shift+Up"}}

	got, ok := hotkeys.Suggest(profiles.Defaults().Hotkeys, "nudge", elsewhere.taken)
	if !ok || got != "Shift+Alt" {
		t.Errorf("got %q, %v; want Shift+Alt, past the far nudges and Ctrl+Alt", got, ok)
	}
}

func TestWithEverythingTakenNothingIsOffered(t *testing.T) {
	everything := func(bindings []hotkeys.Binding) []hotkeys.Binding { return bindings }

	if got, ok := hotkeys.Suggest(profiles.Defaults().Hotkeys, "toggle", everything); ok {
		t.Errorf("got %q, want no suggestion", got)
	}
}

func TestAFieldOffTheListGetsNoSuggestion(t *testing.T) {
	elsewhere := &heldElsewhere{}

	if got, ok := hotkeys.Suggest(profiles.Defaults().Hotkeys, "launch", elsewhere.taken); ok || elsewhere.asked != 0 {
		t.Errorf("got %q, %v after %d questions; want nothing and no question", got, ok, elsewhere.asked)
	}
}
