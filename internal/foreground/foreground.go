package foreground

import (
	"image"

	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

type Window struct {
	Handle    uintptr // zero while Windows passes the focus from one window to the next
	Process   uint32
	Minimized bool
	Client    image.Rectangle // in screen pixels
}

type Process struct {
	ID  uint32
	Exe string
}

type Desktop interface {
	Foreground() Window
	Processes() ([]Process, error)
}

type Target struct {
	Game   profiles.Game
	Client image.Rectangle
}

// Watcher remembers the exe of the window in front, so the process list, the costly part, is
// read again only when another window comes forward.
type Watcher struct {
	desktop Desktop
	own     uint32
	seen    Window
	exe     string
}

func NewWatcher(desktop Desktop, own uint32) *Watcher {
	return &Watcher{desktop: desktop, own: own}
}

func (w *Watcher) Target(settings profiles.Settings) (Target, bool, error) {
	window := w.desktop.Foreground()
	if window.Handle == 0 || window.Process == w.own || window.Minimized || !settings.Visible {
		return Target{}, false, nil
	}

	exe, err := w.exeOf(window)
	if err != nil {
		return Target{}, false, err
	}

	game, ok := settings.Game(exe)
	if !ok || !game.Enabled {
		return Target{}, false, nil
	}

	return Target{Game: game, Client: window.Client}, true, nil
}

func (w *Watcher) exeOf(window Window) (string, error) {
	if window.Handle == w.seen.Handle && window.Process == w.seen.Process {
		return w.exe, nil
	}

	processes, err := w.desktop.Processes()
	if err != nil {
		return "", err
	}

	w.seen, w.exe = window, ""
	for _, p := range processes {
		if p.ID == window.Process {
			w.exe = p.Exe
		}
	}

	return w.exe, nil
}
