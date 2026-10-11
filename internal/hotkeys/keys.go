package hotkeys

import (
	"errors"
	"fmt"
	"slices"
)

type Registrar interface {
	Register(b Binding) error
	Unregister(b Binding) error
}

// Keys holds hotkeys only while a game is in front: Ctrl+Shift with an arrow selects a word in
// any editor, and a program that kept it all the time would take that away.
type Keys struct {
	registrar Registrar
	wanted    []Binding
	held      []Binding
}

func NewKeys(registrar Registrar) *Keys {
	return &Keys{registrar: registrar}
}

// Hold leaves Windows alone when asked for what it was asked for last time, whatever came of
// it then: a hotkey another program holds is tried again when the game next comes forward.
func (k *Keys) Hold(bindings []Binding) error {
	if slices.Equal(bindings, k.wanted) {
		return nil
	}
	k.wanted = bindings

	var errs []error
	for _, b := range k.held {
		errs = append(errs, k.registrar.Unregister(b))
	}
	k.held = nil

	for _, b := range bindings {
		if err := k.registrar.Register(b); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", b.Text, err))
			continue
		}
		k.held = append(k.held, b)
	}

	return errors.Join(errs...)
}
