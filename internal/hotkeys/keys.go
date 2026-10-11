package hotkeys

import (
	"errors"
	"fmt"
	"slices"
	"sync"
)

var ErrTaken = errors.New("hotkeys: another program holds the hotkey")

type Registrar interface {
	Register(b Binding) error
	Unregister(b Binding) error
}

// Keys holds hotkeys only while a game is in front: Ctrl+Shift with an arrow selects a word in
// any editor, and a program that kept it all the time would take that away.
type Keys struct {
	registrar Registrar

	mu     sync.Mutex
	wanted []Binding
	held   []Binding
}

func NewKeys(registrar Registrar) *Keys {
	return &Keys{registrar: registrar}
}

// Hold registers whatever is wanted and not held yet, so a hotkey another program had a moment
// ago is tried again on the next tick, and one already held is left alone.
func (k *Keys) Hold(bindings []Binding) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	var errs []error
	if !slices.Equal(bindings, k.wanted) {
		for _, b := range k.held {
			errs = append(errs, k.registrar.Unregister(b))
		}
		k.held, k.wanted = nil, bindings
	}

	for _, b := range k.wanted {
		if slices.Contains(k.held, b) {
			continue
		}
		if err := k.registrar.Register(b); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", b.Text, err))
			continue
		}
		k.held = append(k.held, b)
	}

	return errors.Join(errs...)
}

// Others leaves out the hotkeys this program holds, so a look for the ones other programs hold
// does not find this program's own.
func (k *Keys) Others(bindings []Binding) []Binding {
	k.mu.Lock()
	defer k.mu.Unlock()

	var others []Binding
	for _, b := range bindings {
		if !slices.ContainsFunc(k.held, b.same) {
			others = append(others, b)
		}
	}

	return others
}
