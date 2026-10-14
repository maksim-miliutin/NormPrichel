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

type Mouse interface {
	Aiming() bool
}

// Loop is what the program does on every tick of the overlay and on every hotkey; all of it
// runs on the thread that owns the windows, so none of it needs a lock.
type Loop struct {
	State   State
	Overlay Overlay
	Games   Games
	Keys    Keys
	Mouse   Mouse

	front    string
	flipped  bool // the toggle hotkey turned the crosshair over until the game leaves the front
	watching bool
}

func (l *Loop) Tick() error {
	current := l.State.Current()
	target, inFront, err := l.Games.Front(current)
	if target.Game.Exe != l.front {
		l.flipped = false
	}
	l.front = target.Game.Exe

	shown := current
	shown.Visible = current.Visible != l.flipped
	l.watching = shown.Visible && target.Game.HideWhileAiming
	if l.watching && l.Mouse.Aiming() {
		shown.Visible = false
	}
	drawn := l.Overlay.Refresh(shown)

	var wanted []hotkeys.Binding
	var unread error
	if inFront {
		wanted, unread = hotkeys.Bindings(current.Hotkeys)
	}

	return errors.Join(drawn, err, unread, l.Keys.Hold(wanted))
}

// Ten looks a second would leave the crosshair over the scope for up to a tenth of a second, so
// while the aim decides, the overlay is asked to look every frame.
func (l *Loop) Watching() bool {
	return l.watching
}

func (l *Loop) Hotkey(action int) error {
	if hotkeys.Toggles(action) {
		l.flipped = !l.flipped
		return nil
	}

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
