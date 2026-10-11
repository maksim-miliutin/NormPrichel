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

func TestATakenHotkeyIsNamedAndTriedAgainUntilItIsFree(t *testing.T) {
	r := &registrar{refused: []string{"Ctrl+Shift+C"}}
	keys := hotkeys.NewKeys(r)
	bindings := defaults(t)

	err := keys.Hold(bindings)
	if !errors.Is(err, errHeld) || !strings.Contains(err.Error(), "Ctrl+Shift+C") {
		t.Errorf("got %v, want the refusal naming Ctrl+Shift+C", err)
	}
	r.take()

	if err := keys.Hold(bindings); !errors.Is(err, errHeld) {
		t.Errorf("while still taken: got %v, want the refusal again", err)
	}
	if got := r.take(); !slices.Equal(got, []string{"+Ctrl+Shift+C"}) {
		t.Errorf("while still taken: made %v, want the taken one tried alone", got)
	}

	r.refused = nil
	if err := keys.Hold(bindings); err != nil {
		t.Errorf("once free: got %v, want it held", err)
	}
	r.take()
	if err := keys.Hold(bindings); err != nil || len(r.take()) != 0 {
		t.Errorf("all held: got %v, want no error and no calls", err)
	}

	keys.Hold(nil)
	if got := r.take(); len(got) != 11 {
		t.Errorf("let go of %v, want all eleven", got)
	}
}

func TestTheHotkeysHeldHereAreNotTakenForOtherPrograms(t *testing.T) {
	r := &registrar{refused: []string{"Ctrl+Shift+C"}}
	keys := hotkeys.NewKeys(r)
	bindings := defaults(t)

	if got := keys.Others(bindings); len(got) != 11 {
		t.Errorf("holding nothing: %d left to look at, want all eleven", len(got))
	}

	keys.Hold(bindings)
	if got := marked("", keys.Others(bindings)); !slices.Equal(got, []string{"Ctrl+Shift+C"}) {
		t.Errorf("holding ten: %v left to look at, want the one held elsewhere", got)
	}

	keys.Hold(nil)
	if got := keys.Others(bindings); len(got) != 11 {
		t.Errorf("after letting go: %d left to look at, want all eleven", len(got))
	}
}

func TestOtherThreadsMayAskWhileHotkeysChange(t *testing.T) {
	keys := hotkeys.NewKeys(&registrar{})
	bindings := defaults(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 200 {
			keys.Others(bindings)
		}
	}()
	for i := range 200 {
		if i%2 == 0 {
			keys.Hold(bindings)
			continue
		}
		keys.Hold(nil)
	}
	<-done
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
