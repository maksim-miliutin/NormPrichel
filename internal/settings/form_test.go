package settings_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/live"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
	"github.com/maksim-miliutin/NormPrichel/internal/settings"
)

// machine stands for what the form asks of Windows: the open windows, the hotkeys other
// programs hold and the start with Windows.
type machine struct {
	windows     []foreground.Choice
	windowFault error
	taken       []string
	starts      bool
	startsFault error
	switchFault error
}

func (m *machine) desktop() ([]foreground.Choice, error) {
	return m.windows, m.windowFault
}

func (m *machine) held(bindings []hotkeys.Binding) []hotkeys.Binding {
	var taken []hotkeys.Binding
	for _, b := range bindings {
		if slices.Contains(m.taken, b.Text) {
			taken = append(taken, b)
		}
	}

	return taken
}

func (m *machine) autostart() (bool, error) {
	return m.starts, m.startsFault
}

func (m *machine) switchAutostart(on bool) error {
	if m.switchFault != nil {
		return m.switchFault
	}
	m.starts = on

	return nil
}

func newForm(t *testing.T, s profiles.Settings) (*settings.Form, *live.State, *machine) {
	t.Helper()

	state := live.New(s, nowhere{}, func(error) {})
	t.Cleanup(state.Close)
	m := &machine{}

	return &settings.Form{
		State:           state,
		Pictures:        profiles.Store{Disk: &disk{files: map[string][]byte{}}, Dir: "NormPrichel"},
		Windows:         m.desktop,
		Taken:           m.held,
		Autostart:       m.autostart,
		SwitchAutostart: m.switchAutostart,
		Version:         "v0.1.0",
		Language:        "ru",
	}, state, m
}

func TestTheFirstGameIsShownUntilAnotherIsChosen(t *testing.T) {
	form, _, _ := newForm(t, listed())

	screen := form.Screen()
	if len(screen.Games) != 2 || !screen.Games[0].Chosen || screen.Games[1].Chosen {
		t.Fatalf("rows %+v, want two with the first chosen", screen.Games)
	}
	if screen.Game == nil || screen.Game.Exe != "stalzone.exe" || screen.Game.Name != "STALZONE" {
		t.Fatalf("shown %+v, want stalzone.exe", screen.Game)
	}

	form.Choose("OTHER.EXE")
	if got := form.Screen(); got.Game.Exe != "other.exe" || !got.Games[1].Chosen || got.Games[0].Chosen {
		t.Errorf("after choosing other.exe the form shows %s with rows %+v", got.Game.Exe, got.Games)
	}

	form.Choose("gone.exe")
	if got := form.Screen().Game.Exe; got != "stalzone.exe" {
		t.Errorf("a choice off the list shows %s, want the first game", got)
	}
}

func TestWithNoGamesThereIsNothingToShow(t *testing.T) {
	form, _, _ := newForm(t, profiles.Defaults())

	if screen := form.Screen(); screen.Game != nil || len(screen.Games) != 0 {
		t.Errorf("rows %+v and game %+v, want none", screen.Games, screen.Game)
	}
}

func TestAGameWithoutANameIsListedByItsExe(t *testing.T) {
	unnamed := listed()
	unnamed.Games[1].Name = ""
	form, _, _ := newForm(t, unnamed)

	if got := form.Screen().Games[1].Name; got != "other.exe" {
		t.Errorf("the row reads %q, want the exe", got)
	}
}

func TestAGameAddedFromAWindowIsShownWithTheCrosshairChosenLast(t *testing.T) {
	form, state, _ := newForm(t, listed())
	form.Choose("other.exe")
	last := state.Last()

	if err := form.AddGame(foreground.Choice{Title: "Блокнот", Exe: "notepad.exe"}); err != nil {
		t.Fatal(err)
	}

	if shown := form.Screen().Game; shown.Exe != "notepad.exe" || shown.Name != "Блокнот" {
		t.Errorf("shown %+v, want the new game", shown)
	}
	if got, _ := state.Current().Game("notepad.exe"); got.Crosshair != last || !got.Enabled {
		t.Errorf("the new game got %+v, want the crosshair chosen last and switched on", got)
	}
}

func TestTheFormOffersTheDesktopWindowsToAdd(t *testing.T) {
	form, _, m := newForm(t, listed())
	m.windows = []foreground.Choice{{Title: "Блокнот", Exe: "notepad.exe"}}

	if got, err := form.WindowsToAdd(); err != nil || !slices.Equal(got, m.windows) {
		t.Errorf("got %+v, %v; want the desktop's windows", got, err)
	}
}

func TestTheShownGameIsRenamedSwitchedOffAndRemoved(t *testing.T) {
	form, state, _ := newForm(t, listed())
	form.Choose("other.exe")

	if err := errors.Join(form.Rename("Другая"), form.Enable(false)); err != nil {
		t.Fatal(err)
	}
	game, _ := state.Current().Game("other.exe")
	if game.Name != "Другая" || game.Enabled {
		t.Errorf("got %+v, want it renamed and off", game)
	}
	if row := form.Screen().Games[1]; row.Name != "Другая" || row.Enabled {
		t.Errorf("the list shows %+v", row)
	}

	if err := form.RemoveGame(); err != nil {
		t.Fatal(err)
	}
	if _, kept := state.Current().Game("other.exe"); kept {
		t.Error("the game is still on the list")
	}
	if got := form.Screen().Game.Exe; got != "stalzone.exe" {
		t.Errorf("after the removal the form shows %s, want the game left", got)
	}
}

func TestARefusalIsWordedUntilAChangeGoesThrough(t *testing.T) {
	form, _, _ := newForm(t, listed())

	err := form.AddGame(foreground.Choice{Title: "STALZONE", Exe: "STALZONE.EXE"})
	if !errors.Is(err, profiles.ErrListed) {
		t.Fatalf("got %v, want ErrListed", err)
	}
	if got, want := form.Screen().Problem, form.Word("error.listed"); got != want || want == "" {
		t.Errorf("the problem reads %q, want %q", got, want)
	}

	if err := form.Rename("S"); err != nil {
		t.Fatal(err)
	}
	if got := form.Screen().Problem; got != "" {
		t.Errorf("after a change went through the problem still reads %q", got)
	}
}

func TestAnUnforeseenFaultIsWordedWithItsDetailAndAReadClearsNothing(t *testing.T) {
	form, _, m := newForm(t, listed())
	form.AddGame(foreground.Choice{Exe: "stalzone.exe"})
	m.windowFault = errors.New("the desktop is gone")

	if _, err := form.WindowsToAdd(); err == nil {
		t.Fatal("no fault")
	}
	if got, want := form.Screen().Problem, form.Word("error.failed")+" the desktop is gone"; got != want {
		t.Errorf("the problem reads %q, want %q", got, want)
	}

	m.windowFault = nil
	form.WindowsToAdd()
	if form.Screen().Problem == "" {
		t.Error("reading the windows again cleared the problem, though nothing changed")
	}
}

func TestTheCrosshairIsSwitchedOffAndOnEverywhere(t *testing.T) {
	form, state, _ := newForm(t, listed())

	form.SetVisible(false)
	if form.Screen().Visible || state.Current().Visible {
		t.Error("the crosshair is still switched on")
	}
	form.SetVisible(true)
	if !form.Screen().Visible || !state.Current().Visible {
		t.Error("the crosshair is still switched off")
	}
}

func TestStartingWithWindowsIsShownAndSwitched(t *testing.T) {
	form, _, m := newForm(t, listed())

	if got := form.Screen().Autostart; got == nil || *got {
		t.Fatalf("got %v, want off", got)
	}
	if err := form.StartWithWindows(true); err != nil || !m.starts {
		t.Errorf("got %v with Windows told %v; want it on", err, m.starts)
	}
	if got := form.Screen().Autostart; got == nil || !*got {
		t.Errorf("got %v, want on", got)
	}

	m.startsFault = errors.New("the registry is locked")
	if got := form.Screen().Autostart; got != nil {
		t.Errorf("got %v, want nothing said while Windows would not tell", *got)
	}

	m.switchFault = errors.New("access denied")
	if err := form.StartWithWindows(false); !errors.Is(err, settings.ErrAutostart) {
		t.Errorf("got %v, want ErrAutostart", err)
	}
	if got, want := form.Screen().Problem, form.Word("error.autostart"); got != want {
		t.Errorf("the problem reads %q, want %q", got, want)
	}
}

func TestSavedCrosshairsAreListedByNameAndTheVersionIsShown(t *testing.T) {
	s := listed()
	s.Saved = []profiles.Saved{{Name: "Зелёный крест", Crosshair: profiles.DefaultCrosshair()}, {Name: "Круг", Crosshair: profiles.DefaultCrosshair()}}
	form, _, _ := newForm(t, s)

	screen := form.Screen()
	if !slices.Equal(screen.Saved, []string{"Зелёный крест", "Круг"}) || screen.Version != "v0.1.0" {
		t.Errorf("saved %q, version %q", screen.Saved, screen.Version)
	}
}
