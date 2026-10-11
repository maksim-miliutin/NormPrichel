package settings

import (
	"errors"
	"fmt"
	"strings"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
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

	chosen     string // the exe on show; one off the list shows the first game
	pictureTab string // the exe whose picture tab is open, even before a picture is added
	problem    error  // the last refusal, nil once a change went through
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
	Picture   bool       // the picture is on show rather than the ready-made shapes
	Knobs     []KnobView // the values the shape is drawn from
	Scale     KnobView
	Offset    profiles.Offset
}

type KnobView struct {
	crosshair.Knob
	Label string
	Value int
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
		screen.Game = f.gameView(shown)
	}
	for _, entry := range current.Saved {
		screen.Saved = append(screen.Saved, entry.Name)
	}

	return screen
}

func (f *Form) gameView(game profiles.Game) *GameView {
	c := game.Crosshair
	view := &GameView{
		Exe:       game.Exe,
		Name:      game.Name,
		Enabled:   game.Enabled,
		Crosshair: c,
		Picture:   c.Image != "" || strings.EqualFold(f.pictureTab, game.Exe),
		Scale:     KnobView{Knob: crosshair.PictureScale(), Label: f.Word("scale"), Value: c.Scale},
		Offset:    game.Offset,
	}
	for _, knob := range crosshair.Knobs(c.Shape) {
		view.Knobs = append(view.Knobs, KnobView{Knob: knob, Label: f.Word(knob.Name), Value: c.Value(knob.Name)})
	}

	return view
}

func (f *Form) Choose(exe string) {
	if !strings.EqualFold(exe, f.shownExe()) {
		f.pictureTab = ""
	}
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

func (f *Form) SetShape(shape crosshair.Shape) error {
	return f.editCrosshair(func(c profiles.Crosshair) (profiles.Crosshair, error) {
		c.Shape = shape
		return c, nil
	})
}

func (f *Form) SetKnob(name string, value int) error {
	return f.editCrosshair(func(c profiles.Crosshair) (profiles.Crosshair, error) {
		style, err := c.WithValue(name, value)
		c.Style = style
		return c, err
	})
}

func (f *Form) SetColor(rgb crosshair.RGB) error {
	return f.editCrosshair(func(c profiles.Crosshair) (profiles.Crosshair, error) {
		c.Color = rgb
		return c, nil
	})
}

func (f *Form) SetOutlineColor(rgb crosshair.RGB) error {
	return f.editCrosshair(func(c profiles.Crosshair) (profiles.Crosshair, error) {
		c.Outline.Color = rgb
		return c, nil
	})
}

func (f *Form) ShowPicture() {
	f.pictureTab = f.shownExe()
}

func (f *Form) ShowReady() error {
	f.pictureTab = ""

	return f.editCrosshair(func(c profiles.Crosshair) (profiles.Crosshair, error) {
		c.Image = ""
		return c, nil
	})
}

func (f *Form) AddPicture(file []byte) error {
	name, err := f.Pictures.AddPicture(file)
	if err != nil {
		f.problem = err
		return err
	}
	f.pictureTab = f.shownExe()

	return f.editCrosshair(func(c profiles.Crosshair) (profiles.Crosshair, error) {
		c.Image = name
		return c, nil
	})
}

func (f *Form) SetScale(percent int) error {
	scale := crosshair.PictureScale()

	return f.editCrosshair(func(c profiles.Crosshair) (profiles.Crosshair, error) {
		c.Scale = min(max(percent, scale.Min), scale.Max)
		return c, nil
	})
}

func (f *Form) editCrosshair(change func(profiles.Crosshair) (profiles.Crosshair, error)) error {
	exe := f.shownExe()

	return f.change(func(s profiles.Settings) (profiles.Settings, error) {
		game, _ := s.Game(exe)
		c, err := change(game.Crosshair)
		if err != nil {
			return s, err
		}
		return s.Edited(exe, func(g profiles.Game) profiles.Game {
			g.Crosshair = c
			return g
		})
	})
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
