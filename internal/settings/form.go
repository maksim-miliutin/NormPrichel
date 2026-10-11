package settings

import (
	"errors"
	"fmt"

	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

type Form struct {
	State           State
	Pictures        Pictures
	Windows         func() ([]foreground.Choice, error)
	Taken           func(bindings []hotkeys.Binding) []hotkeys.Binding
	Autostart       func() (bool, error)
	SwitchAutostart func(on bool) error
	Version         string
	Language        string

	chosen  string // the exe on show; one off the list shows the first game
	problem error  // the last refusal, nil once a change went through
}

type Screen struct {
	Visible   bool
	Autostart *bool // nil when Windows would not tell
	Games     []GameRow
	Game      *GameView // nil while the list is empty
	Saved     []string
	Problem   string
	Version   string
}

type GameRow struct {
	Exe     string
	Name    string
	Enabled bool
	Chosen  bool
}

type GameView struct {
	Exe       string
	Name      string
	Enabled   bool
	Crosshair profiles.Crosshair
	Offset    profiles.Offset
}

func (f *Form) Word(key string) string {
	return textFor(f.Language)[key]
}

func (f *Form) Screen() Screen {
	current := f.State.Current()
	screen := Screen{Visible: current.Visible, Autostart: f.starts(), Problem: f.worded(f.problem), Version: f.Version}

	shown, ok := f.shown(current)
	for _, game := range current.Games {
		screen.Games = append(screen.Games, GameRow{Exe: game.Exe, Name: titleOf(game), Enabled: game.Enabled, Chosen: ok && game.Exe == shown.Exe})
	}
	if ok {
		screen.Game = &GameView{Exe: shown.Exe, Name: shown.Name, Enabled: shown.Enabled, Crosshair: shown.Crosshair, Offset: shown.Offset}
	}
	for _, entry := range current.Saved {
		screen.Saved = append(screen.Saved, entry.Name)
	}

	return screen
}

func (f *Form) Choose(exe string) {
	f.chosen = exe
}

// Reading changes nothing, so it leaves a standing refusal on the screen and only adds its own.
func (f *Form) WindowsToAdd() ([]foreground.Choice, error) {
	choices, err := f.Windows()
	if err != nil {
		f.problem = err
	}

	return choices, err
}

func (f *Form) AddGame(choice foreground.Choice) error {
	last := f.State.Last()
	err := f.change(func(s profiles.Settings) (profiles.Settings, error) {
		return s.WithGame(profiles.Game{Exe: choice.Exe, Name: choice.Title, Enabled: true, Crosshair: last})
	})
	if err == nil {
		f.chosen = choice.Exe
	}

	return err
}

func (f *Form) Rename(name string) error {
	return f.edit(func(g profiles.Game) profiles.Game {
		g.Name = name
		return g
	})
}

func (f *Form) Enable(on bool) error {
	return f.edit(func(g profiles.Game) profiles.Game {
		g.Enabled = on
		return g
	})
}

func (f *Form) RemoveGame() error {
	exe := f.shownExe()

	return f.change(func(s profiles.Settings) (profiles.Settings, error) {
		return s.WithoutGame(exe)
	})
}

func (f *Form) SetVisible(on bool) error {
	return f.change(func(s profiles.Settings) (profiles.Settings, error) {
		s.Visible = on
		return s, nil
	})
}

func (f *Form) StartWithWindows(on bool) error {
	err := f.SwitchAutostart(on)
	if err != nil {
		err = fmt.Errorf("%w: %w", ErrAutostart, err)
	}
	f.problem = err

	return err
}

func (f *Form) edit(change func(profiles.Game) profiles.Game) error {
	exe := f.shownExe()

	return f.change(func(s profiles.Settings) (profiles.Settings, error) {
		return s.Edited(exe, change)
	})
}

func (f *Form) change(edit func(profiles.Settings) (profiles.Settings, error)) error {
	_, err := f.State.Change(edit)
	f.problem = err

	return err
}

func (f *Form) shown(current profiles.Settings) (profiles.Game, bool) {
	if game, ok := current.Game(f.chosen); ok {
		return game, true
	}
	if len(current.Games) == 0 {
		return profiles.Game{}, false
	}

	return current.Games[0], true
}

func (f *Form) shownExe() string {
	game, _ := f.shown(f.State.Current())

	return game.Exe
}

func (f *Form) starts() *bool {
	on, err := f.Autostart()
	if err != nil {
		return nil
	}

	return &on
}

func (f *Form) worded(err error) string {
	if err == nil {
		return ""
	}
	for _, reason := range reasons {
		if errors.Is(err, reason.err) {
			return f.Word("error." + reason.code)
		}
	}

	return f.Word("error.failed") + " " + err.Error()
}

func titleOf(game profiles.Game) string {
	if game.Name == "" {
		return game.Exe
	}

	return game.Name
}
