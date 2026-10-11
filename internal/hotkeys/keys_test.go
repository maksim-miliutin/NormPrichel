package hotkeys_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var errHeld = errors.New("held by another program")

// registrar writes down what Windows was asked to do: "+text" to register, "-text" to let go;
// texts in refused fail to register.
type registrar struct {
	calls   []string
	refused []string
}

func (r *registrar) Register(b hotkeys.Binding) error {
	r.calls = append(r.calls, "+"+b.Text)
	if slices.Contains(r.refused, b.Text) {
		return errHeld
	}

	return nil
}

func (r *registrar) Unregister(b hotkeys.Binding) error {
	r.calls = append(r.calls, "-"+b.Text)

	return nil
}

func (r *registrar) take() []string {
	calls := r.calls
	r.calls = nil

	return calls
}

func defaults(t *testing.T) []hotkeys.Binding {
	t.Helper()

	bindings, err := hotkeys.Bindings(profiles.Defaults().Hotkeys)
	if err != nil {
		t.Fatal(err)
	}

	return bindings
}

func marked(sign string, bindings []hotkeys.Binding) []string {
	texts := make([]string, len(bindings))
	for i, b := range bindings {
		texts[i] = sign + b.Text
	}

	return texts
}

func TestHotkeysAreHeldWhileWantedAndLetGoAfter(t *testing.T) {
	r := &registrar{}
	keys := hotkeys.NewKeys(r)
	bindings := defaults(t)

	if err := keys.Hold(bindings); err != nil {
		t.Fatal(err)
	}
	if got := r.take(); !slices.Equal(got, marked("+", bindings)) {
		t.Errorf("registered %v, want all eleven", got)
	}

	if err := keys.Hold(nil); err != nil {
		t.Fatal(err)
	}
	if got := r.take(); !slices.Equal(got, marked("-", bindings)) {
		t.Errorf("let go of %v, want all eleven", got)
	}
}

func TestAskingForTheSameHotkeysAgainTouchesNothing(t *testing.T) {
	r := &registrar{}
	keys := hotkeys.NewKeys(r)

	keys.Hold(defaults(t))
	r.take()
	for range 3 {
		if err := keys.Hold(defaults(t)); err != nil {
			t.Fatal(err)
		}
	}
	keys.Hold(nil)
	r.take()
	keys.Hold(nil)

	if got := r.take(); len(got) != 0 {
		t.Errorf("asking again made %v, want no calls", got)
	}
}

func TestATakenHotkeyIsNamedAndTriedAgainOnlyWhenAskedAnew(t *testing.T) {
	r := &registrar{refused: []string{"Ctrl+Shift+C"}}
	keys := hotkeys.NewKeys(r)
	bindings := defaults(t)

	err := keys.Hold(bindings)
	if !errors.Is(err, errHeld) || !strings.Contains(err.Error(), "Ctrl+Shift+C") {
		t.Errorf("got %v, want the refusal naming Ctrl+Shift+C", err)
	}
	r.take()

	if err := keys.Hold(bindings); err != nil || len(r.take()) != 0 {
		t.Errorf("the same hotkeys again: got %v, want no error and no retry", err)
	}

	keys.Hold(nil)
	if got := r.take(); slices.Contains(got, "-Ctrl+Shift+C") || len(got) != 10 {
		t.Errorf("let go of %v, want the ten that were held and not the refused one", got)
	}

	keys.Hold(bindings)
	if got := r.take(); !slices.Equal(got, marked("+", bindings)) {
		t.Errorf("asked anew: registered %v, want all eleven tried again", got)
	}
}

func TestChangedHotkeysReplaceTheOldOnes(t *testing.T) {
	r := &registrar{}
	keys := hotkeys.NewKeys(r)
	old := defaults(t)
	changed := profiles.Defaults().Hotkeys
	changed.Toggle = "Ctrl+Alt+F9"
	fresh, err := hotkeys.Bindings(changed)
	if err != nil {
		t.Fatal(err)
	}

	keys.Hold(old)
	r.take()
	keys.Hold(fresh)

	if got := r.take(); !slices.Equal(got, append(marked("-", old), marked("+", fresh)...)) {
		t.Errorf("made %v, want the old eleven let go and the new eleven held", got)
	}
}
