package hotkeys_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

func crosshairOf(shape crosshair.Shape) profiles.Crosshair {
	c := profiles.DefaultCrosshair()
	c.Shape = shape

	return c
}

func listed() profiles.Settings {
	settings := profiles.Defaults()
	settings.Games = []profiles.Game{
		{Exe: "stalzone.exe", Enabled: true, Crosshair: crosshairOf(crosshair.Dot)},
		{Exe: "other.exe", Enabled: true, Crosshair: crosshairOf(crosshair.Cross), Offset: profiles.Offset{X: 4, Y: -2}},
	}
	settings.Saved = []profiles.Saved{
		{Name: "dot", Crosshair: crosshairOf(crosshair.Dot)},
		{Name: "cross", Crosshair: crosshairOf(crosshair.Cross)},
	}

	return settings
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

func TestTheToggleIsToldApartAndChangesNoSetting(t *testing.T) {
	toggle := actionOf(t, "Ctrl+Shift+X")

	if !hotkeys.Toggles(toggle) {
		t.Error("the toggle is not told apart")
	}
	for _, other := range []int{actionOf(t, "Ctrl+Shift+Home"), actionOf(t, "Ctrl+Shift+Left"), -1, 99} {
		if hotkeys.Toggles(other) {
			t.Errorf("action %d is taken for the toggle", other)
		}
	}

	got, err := hotkeys.Apply(listed(), toggle, "stalzone.exe")
	if err != nil || !reflect.DeepEqual(got, listed()) {
		t.Errorf("got %+v, %v; want the settings as they were", got, err)
	}
}

func TestArrowsNudgeTheGameInFrontByOneOrTenPixels(t *testing.T) {
	steps := map[string]profiles.Offset{
		"Ctrl+Shift+Left":      {X: -1},
		"Ctrl+Shift+Right":     {X: 1},
		"Ctrl+Shift+Up":        {Y: -1},
		"Ctrl+Shift+Down":      {Y: 1},
		"Ctrl+Shift+Alt+Left":  {X: -10},
		"Ctrl+Shift+Alt+Right": {X: 10},
		"Ctrl+Shift+Alt+Up":    {Y: -10},
		"Ctrl+Shift+Alt+Down":  {Y: 10},
	}

	for text, step := range steps {
		got, err := hotkeys.Apply(listed(), actionOf(t, text), "OTHER.EXE")
		if err != nil {
			t.Errorf("%s: %v", text, err)
			continue
		}

		if want := (profiles.Offset{X: 4 + step.X, Y: -2 + step.Y}); got.Games[1].Offset != want {
			t.Errorf("%s: offset %+v, want %+v", text, got.Games[1].Offset, want)
		}
		if got.Games[0].Offset != (profiles.Offset{}) {
			t.Errorf("%s: moved the game behind, to %+v", text, got.Games[0].Offset)
		}
	}
}

func TestHomeBringsTheCrosshairBackToTheCentre(t *testing.T) {
	got, err := hotkeys.Apply(listed(), actionOf(t, "Ctrl+Shift+Home"), "other.exe")
	if err != nil || got.Games[1].Offset != (profiles.Offset{}) {
		t.Errorf("got %+v, %v; want the offset at zero", got.Games[1].Offset, err)
	}
}

func TestTheNextSavedCrosshairComesForTheGameInFront(t *testing.T) {
	next := actionOf(t, "Ctrl+Shift+C")

	once, err := hotkeys.Apply(listed(), next, "other.exe")
	if err != nil || once.Games[1].Crosshair.Shape != crosshair.Dot {
		t.Fatalf("got %q, %v; want the dot after the cross, round the list", once.Games[1].Crosshair.Shape, err)
	}
	if once.Games[0].Crosshair.Shape != crosshair.Dot {
		t.Errorf("the game behind changed to %q", once.Games[0].Crosshair.Shape)
	}

	twice, err := hotkeys.Apply(once, next, "other.exe")
	if err != nil || twice.Games[1].Crosshair.Shape != crosshair.Cross {
		t.Errorf("got %q, %v; want the cross again", twice.Games[1].Crosshair.Shape, err)
	}
}

func TestKeysForAGameOffTheListChangeNothing(t *testing.T) {
	for _, text := range []string{"Ctrl+Shift+Left", "Ctrl+Shift+Home", "Ctrl+Shift+C"} {
		if _, err := hotkeys.Apply(listed(), actionOf(t, text), "notepad.exe"); !errors.Is(err, profiles.ErrNoGame) {
			t.Errorf("%s: got %v, want ErrNoGame", text, err)
		}
	}
}

func TestAnActionNumberOffTheTableIsRefused(t *testing.T) {
	bindings, err := hotkeys.Bindings(profiles.Defaults().Hotkeys)
	if err != nil {
		t.Fatal(err)
	}

	for _, action := range []int{-1, len(bindings)} {
		if _, err := hotkeys.Apply(listed(), action, "other.exe"); !errors.Is(err, hotkeys.ErrNoAction) {
			t.Errorf("action %d: got %v, want ErrNoAction", action, err)
		}
	}
}
