package app_test

import (
	"errors"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/app"
	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/live"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var (
	errScreen = errors.New("the screen refused")
	errHeld   = errors.New("held elsewhere")
)

type overlay struct {
	drawn []profiles.Settings
	fail  error
}

func (o *overlay) Refresh(settings profiles.Settings) error {
	o.drawn = append(o.drawn, settings)

	return o.fail
}

// games puts one exe in front, or nothing when exe is empty.
type games struct {
	exe string
}

func (g *games) Front(settings profiles.Settings) (foreground.Target, bool, error) {
	game, ok := settings.Game(g.exe)

	return foreground.Target{Game: game}, ok, nil
}

type keys struct {
	held [][]hotkeys.Binding
	fail error
}

func (k *keys) Hold(bindings []hotkeys.Binding) error {
	k.held = append(k.held, bindings)

	return k.fail
}

func (k *keys) last() []hotkeys.Binding {
	return k.held[len(k.held)-1]
}

type nowhere struct{}

func (nowhere) Save(profiles.Settings) error { return nil }

func settings() profiles.Settings {
	s := profiles.Defaults()
	s.Games = []profiles.Game{
		{Exe: "stalzone.exe", Enabled: true, Crosshair: profiles.DefaultCrosshair()},
		{Exe: "other.exe", Enabled: true, Crosshair: profiles.DefaultCrosshair()},
	}

	return s
}

type world struct {
	loop    *app.Loop
	state   *live.State
	overlay *overlay
	games   *games
	keys    *keys
}

func newWorld(t *testing.T, s profiles.Settings) world {
	t.Helper()

	w := world{state: live.New(s, nowhere{}, func(error) {}), overlay: &overlay{}, games: &games{}, keys: &keys{}}
	t.Cleanup(w.state.Close)
	w.loop = &app.Loop{State: w.state, Overlay: w.overlay, Games: w.games, Keys: w.keys}

	return w
}

func actionOf(t *testing.T, text string) int {
	t.Helper()

	bindings, err := hotkeys.Bindings(profiles.Defaults().Hotkeys)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range bindings {
		if b.Text == text {
			return b.Action
		}
	}
	t.Fatalf("nothing is bound to %s", text)

	return 0
}

func TestHotkeysAreHeldOnlyWhileAGameIsInFront(t *testing.T) {
	w := newWorld(t, settings())

	w.games.exe = "stalzone.exe"
	if err := w.loop.Tick(); err != nil {
		t.Fatal(err)
	}
	if got := len(w.keys.last()); got != 11 {
		t.Errorf("with the game in front %d hotkeys are held, want 11", got)
	}

	w.games.exe = ""
	if err := w.loop.Tick(); err != nil {
		t.Fatal(err)
	}
	if got := w.keys.last(); got != nil {
		t.Errorf("with nothing in front %d hotkeys are held, want none", len(got))
	}
}

func TestAHotkeyActsOnTheGameInFrontAtTheLastTick(t *testing.T) {
	w := newWorld(t, settings())
	w.games.exe = "other.exe"
	w.loop.Tick()

	if err := w.loop.Hotkey(actionOf(t, "Ctrl+Shift+Alt+Up")); err != nil {
		t.Fatal(err)
	}

	got := w.state.Current()
	if got.Games[1].Offset != (profiles.Offset{Y: -10}) || got.Games[0].Offset != (profiles.Offset{}) {
		t.Errorf("offsets %+v and %+v, want other.exe ten pixels up and stalzone.exe where it was", got.Games[1].Offset, got.Games[0].Offset)
	}
}

func TestTheOverlayIsDrawnFromTheSettingsAsTheyAreNow(t *testing.T) {
	w := newWorld(t, settings())
	w.games.exe = "stalzone.exe"
	w.loop.Tick()

	w.loop.Hotkey(actionOf(t, "Ctrl+Shift+Right"))
	w.loop.Tick()

	last := w.overlay.drawn[len(w.overlay.drawn)-1]
	if last.Games[0].Offset.X != 1 {
		t.Errorf("the overlay was last drawn at %+v, want one pixel right", last.Games[0].Offset)
	}
}

func TestTheToggleWorksWithNoGameInFront(t *testing.T) {
	w := newWorld(t, settings())

	if err := w.loop.Toggle(); err != nil || w.state.Current().Visible {
		t.Fatalf("got %v; want the crosshair hidden", err)
	}
	if err := w.loop.Toggle(); err != nil || !w.state.Current().Visible {
		t.Errorf("got %v; want the crosshair shown again", err)
	}
}

func TestHotkeysThatDoNotReadHoldNothingAndSaySo(t *testing.T) {
	broken := settings()
	broken.Hotkeys.Toggle = "Shift+X"
	w := newWorld(t, broken)
	w.games.exe = "stalzone.exe"

	err := w.loop.Tick()
	if !errors.Is(err, hotkeys.ErrBadHotkey) {
		t.Errorf("got %v, want ErrBadHotkey", err)
	}
	if got := w.keys.last(); got != nil {
		t.Errorf("%d hotkeys are held, want none", len(got))
	}
}

func TestEveryFaultOfATickComesBackTogether(t *testing.T) {
	w := newWorld(t, settings())
	w.games.exe = "stalzone.exe"
	w.overlay.fail = errScreen
	w.keys.fail = errHeld

	err := w.loop.Tick()
	if !errors.Is(err, errScreen) || !errors.Is(err, errHeld) {
		t.Errorf("got %v, want both the screen and the hotkey faults", err)
	}
}

func TestAHotkeyThatArrivesAfterTheGameLeftChangesNothing(t *testing.T) {
	w := newWorld(t, settings())
	w.games.exe = "stalzone.exe"
	w.loop.Tick()
	w.games.exe = ""
	w.loop.Tick()

	err := w.loop.Hotkey(actionOf(t, "Ctrl+Shift+Left"))
	if !errors.Is(err, profiles.ErrNoGame) || w.state.Current().Games[0].Offset != (profiles.Offset{}) {
		t.Errorf("got %v with offset %+v; want ErrNoGame and the game left where it was", err, w.state.Current().Games[0].Offset)
	}
}

func TestAHotkeyForAGameThatLeftChangesNothing(t *testing.T) {
	w := newWorld(t, settings())
	w.games.exe = "stalzone.exe"
	w.loop.Tick()
	w.state.Change(func(s profiles.Settings) (profiles.Settings, error) { return s.WithoutGame("stalzone.exe") })

	if err := w.loop.Hotkey(actionOf(t, "Ctrl+Shift+Home")); !errors.Is(err, profiles.ErrNoGame) {
		t.Errorf("got %v, want ErrNoGame", err)
	}
}
