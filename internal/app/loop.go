package app

import (
	"errors"

	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

type State interface {
	Current() profiles.Settings
	Change(edit func(profiles.Settings) (profiles.Settings, error)) (profiles.Settings, error)
}

type Overlay interface {
	Refresh(settings profiles.Settings) error
}

type Games interface {
	Front(settings profiles.Settings) (foreground.Target, bool, error)
}

type Keys interface {
	Hold(bindings []hotkeys.Binding) error
}

// Loop is what the program does on every tick of the overlay and on every hotkey; all of it
// runs on the thread that owns the windows, so none of it needs a lock.
type Loop struct {
	State   State
	Overlay Overlay
	Games   Games
	Keys    Keys

	front string
}

func (l *Loop) Tick() error {
	current := l.State.Current()
	drawn := l.Overlay.Refresh(current)

	target, inFront, err := l.Games.Front(current)
	l.front = target.Game.Exe

	var wanted []hotkeys.Binding
	var unread error
	if inFront {
		wanted, unread = hotkeys.Bindings(current.Hotkeys)
	}

	return errors.Join(drawn, err, unread, l.Keys.Hold(wanted))
}

func (l *Loop) Hotkey(action int) error {
	_, err := l.State.Change(func(s profiles.Settings) (profiles.Settings, error) {
		return hotkeys.Apply(s, action, l.front)
	})

	return err
}

func (l *Loop) Toggle() error {
	_, err := l.State.Change(func(s profiles.Settings) (profiles.Settings, error) {
		s.Visible = !s.Visible
		return s, nil
	})

	return err
}
