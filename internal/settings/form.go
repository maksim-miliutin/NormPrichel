package settings

import (
	"errors"
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var (
	ErrAutostart = errors.New("settings: starting with Windows cannot be switched")
	ErrNoCode    = errors.New("settings: a picture of one's own has no code")
)

type State interface {
	Current() profiles.Settings
	Last() profiles.Crosshair
	Change(edit func(profiles.Settings) (profiles.Settings, error)) (profiles.Settings, error)
	Reset() (profiles.Settings, error)
}

type Pictures interface {
	AddPicture(file []byte) (string, error)
	Paint(c profiles.Crosshair) (*image.RGBA, error)
}

type Form struct {
	State           State
	Pictures        Pictures
	Windows         func() ([]foreground.Choice, error)
	Taken           func(bindings []hotkeys.Binding) []hotkeys.Binding
	Autostart       func() (bool, error)
	SwitchAutostart func(on bool) error
	Version         string
	WindowsLanguage string

	chosen     string // the exe on show; one off the list shows the first game
	pictureTab string // the exe whose picture tab is open, even before a picture is added
	far        bool   // the arrows move the crosshair ten pixels rather than one
	problem    error  // the last refusal, nil once a change went through
	copied     bool   // the code on show went to the clipboard and nothing has changed since
	painted    *painted
}

type Screen struct {
	Visible   bool
	Autostart *bool // nil when Windows would not tell
	Games     []GameRow
	Game      *GameView // nil while the list is empty
	Saved     []string
	FarStep   bool
	Hotkeys   []HotkeyRow
	Problem   string
	Version   string
	Language  string
	Copied    bool
}

type HotkeyRow struct {
	Field      string
	Label      string
	Text       string
	Taken      string // the hotkey another program holds; empty while it is free
	Suggestion string // a free one to take instead; empty when none is free
}

type GameRow struct {
	Exe     string
	Name    string
	Enabled bool
	Chosen  bool
}

type GameView struct {
	Exe             string
	Name            string
	Enabled         bool
	Crosshair       profiles.Crosshair
	Picture         bool       // the picture is on show rather than the ready-made shapes
	Knobs           []KnobView // the values the shape is drawn from
	Scale           KnobView
	Offset          profiles.Offset
	HideWhileAiming bool
}

type KnobView struct {
	crosshair.Knob
	Label string
	Value int
}

func (f *Form) Word(key string) string {
	return Word(Language(f.State.Current().Language, f.WindowsLanguage), key)
}

func (f *Form) Screen() Screen {
	current := f.State.Current()
	screen := Screen{
		Visible:   current.Visible,
		Autostart: f.starts(),
		FarStep:   f.far,
		Version:   f.Version,
		Language:  Language(current.Language, f.WindowsLanguage),
		Copied:    f.copied,
	}

	shown, ok := f.shown(current)
	problem := f.problem
	if problem == nil && ok {
		_, problem = f.paint(shown.Crosshair)
	}
	screen.Problem = f.worded(problem)
	for _, game := range current.Games {
		screen.Games = append(screen.Games, GameRow{Exe: game.Exe, Name: titleOf(game), Enabled: game.Enabled, Chosen: ok && game.Exe == shown.Exe})
	}
	if ok {
		screen.Game = f.gameView(shown)
	}
	for _, entry := range current.Saved {
		screen.Saved = append(screen.Saved, entry.Name)
	}
	screen.Hotkeys = f.hotkeyRows(current.Hotkeys)

	return screen
}

// Each field is named once, by the first of its hotkeys that another program holds.
func (f *Form) hotkeyRows(keys profiles.Hotkeys) []HotkeyRow {
	taken := map[string]string{}
	bindings, _ := hotkeys.Bindings(keys)
	for _, b := range f.Taken(bindings) {
		if _, named := taken[b.Field]; !named {
			taken[b.Field] = b.Text
		}
	}

	var rows []HotkeyRow
	for _, field := range hotkeys.Fields() {
		row := HotkeyRow{Field: field, Label: f.Word("hotkey." + field), Text: hotkeys.Get(keys, field), Taken: taken[field]}
		if row.Taken != "" {
			row.Suggestion, _ = hotkeys.Suggest(keys, field, f.Taken)
		}
		rows = append(rows, row)
	}

	return rows
}

func (f *Form) gameView(game profiles.Game) *GameView {
	c := game.Crosshair
	view := &GameView{
		Exe:             game.Exe,
		Name:            game.Name,
		Enabled:         game.Enabled,
		Crosshair:       c,
		Picture:         c.Image != "" || strings.EqualFold(f.pictureTab, game.Exe),
		Scale:           KnobView{Knob: crosshair.PictureScale(), Label: f.Word("scale"), Value: c.Scale},
		Offset:          game.Offset,
		HideWhileAiming: game.HideWhileAiming,
	}
	for _, knob := range crosshair.Knobs(c.Shape) {
		view.Knobs = append(view.Knobs, KnobView{Knob: knob, Label: f.Word(knob.Name), Value: c.Value(knob.Name)})
	}

	return view
}

func (f *Form) Choose(exe string) {
	if !strings.EqualFold(exe, f.shownExe()) {
		f.pictureTab, f.copied = "", false
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

func (f *Form) HideWhileAiming(on bool) error {
	return f.edit(func(g profiles.Game) profiles.Game {
		g.HideWhileAiming = on
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

func (f *Form) SetLanguage(language string) error {
	return f.change(func(s profiles.Settings) (profiles.Settings, error) {
		s.Language = language
		return s, nil
	})
}

func (f *Form) ResetAll() error {
	_, err := f.State.Reset()
	if err == nil {
		f.chosen, f.pictureTab, f.far, f.painted = "", "", false, nil
	}
	f.problem = err

	return err
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

func (f *Form) ResetCrosshair() error {
	f.pictureTab = ""

	return f.editCrosshair(func(profiles.Crosshair) (profiles.Crosshair, error) {
		return profiles.DefaultCrosshair(), nil
	})
}

func (f *Form) CopyCode(clipboard func(string) error) error {
	game, ok := f.shown(f.State.Current())
	if !ok || game.Crosshair.Image != "" {
		f.problem = ErrNoCode
		return ErrNoCode
	}

	f.problem = clipboard(game.Crosshair.Code())
	f.copied = f.problem == nil

	return f.problem
}

func (f *Form) UseCode(text string) error {
	style, err := crosshair.ParseCode(text)
	if err != nil {
		f.problem = err
		return err
	}
	f.pictureTab = ""

	return f.editCrosshair(func(c profiles.Crosshair) (profiles.Crosshair, error) {
		c.Style, c.Image = style, ""
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
	f.painted = nil

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

func (f *Form) SetOffset(x, y int) error {
	return f.edit(func(g profiles.Game) profiles.Game {
		g.Offset = profiles.Offset{X: x, Y: y}
		return g
	})
}

func (f *Form) SetFarStep(far bool) {
	f.far = far
}

func (f *Form) Nudge(dx, dy int) error {
	step := 1
	if f.far {
		step = 10
	}

	return f.edit(func(g profiles.Game) profiles.Game {
		g.Offset = profiles.Offset{X: g.Offset.X + dx*step, Y: g.Offset.Y + dy*step}
		return g
	})
}

func (f *Form) Centre() error {
	return f.SetOffset(0, 0)
}

// A float past the range of int converts to garbage, so it is held near the offset limit first.
func Whole(text string) (int, bool) {
	value, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}

	return int(math.Round(min(max(value, -1e6), 1e6))), true
}

func (f *Form) Save(name string) error {
	name = strings.TrimSpace(name)
	game, ok := f.shown(f.State.Current())
	if name == "" || !ok {
		return nil
	}

	return f.change(func(s profiles.Settings) (profiles.Settings, error) {
		return s.WithSaved(profiles.Saved{Name: name, Crosshair: game.Crosshair}), nil
	})
}

func (f *Form) UseSaved(index int) error {
	exe := f.shownExe()
	f.pictureTab = ""

	return f.change(func(s profiles.Settings) (profiles.Settings, error) {
		if index < 0 || index >= len(s.Saved) {
			return s, fmt.Errorf("%w: %d", profiles.ErrNoSaved, index)
		}
		entry := s.Saved[index].Crosshair
		return s.Edited(exe, func(g profiles.Game) profiles.Game {
			g.Crosshair = entry
			return g
		})
	})
}

func (f *Form) ForgetSaved(index int) error {
	return f.change(func(s profiles.Settings) (profiles.Settings, error) {
		return s.WithoutSaved(index)
	})
}

func (f *Form) SetHotkey(field, text string) error {
	return f.change(func(s profiles.Settings) (profiles.Settings, error) {
		keys, err := hotkeys.Set(s.Hotkeys, field, text)
		if err != nil {
			return s, err
		}
		if _, err := hotkeys.Bindings(keys); err != nil {
			return s, err
		}
		s.Hotkeys = keys
		return s, nil
	})
}

func (f *Form) TakeSuggestion(field string) error {
	for _, row := range f.hotkeyRows(f.State.Current().Hotkeys) {
		if row.Field == field && row.Suggestion != "" {
			return f.SetHotkey(field, row.Suggestion)
		}
	}

	return nil
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
	f.problem, f.copied = err, false

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
