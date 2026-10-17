package settings_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/png"
	"slices"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
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
		WindowsLanguage: "ru",
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

func TestHidingWhileAimingIsSwitchedForTheShownGameAlone(t *testing.T) {
	form, state, _ := newForm(t, listed())
	form.Choose("other.exe")

	if err := form.HideWhileAiming(true); err != nil {
		t.Fatal(err)
	}
	other, _ := state.Current().Game("other.exe")
	stalzone, _ := state.Current().Game("stalzone.exe")
	if !other.HideWhileAiming || stalzone.HideWhileAiming {
		t.Errorf("hiding while aiming is %v for the shown game and %v for the other, want on and off", other.HideWhileAiming, stalzone.HideWhileAiming)
	}
	if !form.Screen().Game.HideWhileAiming {
		t.Error("the form shows the shown game without hiding while aiming")
	}

	if err := form.HideWhileAiming(false); err != nil {
		t.Fatal(err)
	}
	if form.Screen().Game.HideWhileAiming {
		t.Error("switched off, hiding while aiming still shows as on")
	}
}

func TestTheChosenLanguageWinsOverTheOneWindowsSpeaks(t *testing.T) {
	for _, c := range []struct{ chosen, windows, want string }{
		{"", "ru", "ru"},
		{"", "en", "en"},
		{"en", "ru", "en"},
		{"ru", "en", "ru"},
		{"de", "ru", "ru"},
	} {
		if got := settings.Language(c.chosen, c.windows); got != c.want {
			t.Errorf("chosen %q with Windows in %q speaks %q, want %q", c.chosen, c.windows, got, c.want)
		}
	}
	if settings.Word("de", "visible") != settings.Word("en", "visible") {
		t.Error("a language without words does not fall back to English")
	}
}

func TestTheFormSpeaksTheChosenLanguageEverywhere(t *testing.T) {
	form, state, _ := newForm(t, listed())
	if got := form.Screen().Language; got != "ru" {
		t.Fatalf("before a choice the form speaks %q, want Windows' ru", got)
	}

	if err := form.SetLanguage("en"); err != nil {
		t.Fatal(err)
	}
	if got := state.Current().Language; got != "en" {
		t.Errorf("the settings keep %q, want en", got)
	}
	screen := form.Screen()
	knob := screen.Game.Knobs[0]
	for _, c := range []struct{ got, key string }{
		{form.Word("visible"), "visible"},
		{knob.Label, knob.Name},
		{screen.Hotkeys[0].Label, "hotkey." + screen.Hotkeys[0].Field},
	} {
		if c.got != settings.Word("en", c.key) || c.got == settings.Word("ru", c.key) {
			t.Errorf("%q reads %q, want the English %q", c.key, c.got, settings.Word("en", c.key))
		}
	}
	if screen.Language != "en" {
		t.Errorf("the screen says the form speaks %q, want en", screen.Language)
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

func knobNames(knobs []settings.KnobView) []string {
	var got []string
	for _, k := range knobs {
		got = append(got, k.Name)
	}

	return got
}

func TestTheShownGameOffersTheKnobsOfItsShapeInWords(t *testing.T) {
	form, _, _ := newForm(t, listed())

	shown := form.Screen().Game
	if got := knobNames(shown.Knobs); !slices.Equal(got, []string{"opacity", "dot", "outline"}) {
		t.Fatalf("a dot offers %v", got)
	}
	for _, k := range shown.Knobs {
		if k.Label != form.Word(k.Name) || k.Label == "" || k.Value != shown.Crosshair.Value(k.Name) || k.Max <= k.Min {
			t.Errorf("knob %+v does not match its word, value or range", k)
		}
	}
}

func TestAnotherShapeOffersItsOwnKnobs(t *testing.T) {
	form, state, _ := newForm(t, listed())

	if err := form.SetShape(crosshair.Cross); err != nil {
		t.Fatal(err)
	}
	if got := knobNames(form.Screen().Game.Knobs); !slices.Equal(got, []string{"opacity", "length", "thickness", "gap", "outline"}) {
		t.Errorf("a cross offers %v", got)
	}
	if game, _ := state.Current().Game("stalzone.exe"); game.Crosshair.Shape != crosshair.Cross {
		t.Errorf("the settings hold %s, want cross", game.Crosshair.Shape)
	}
}

func TestAKnobIsSetWithinItsRangeForTheShownGameAlone(t *testing.T) {
	form, state, _ := newForm(t, listed())
	form.Choose("other.exe")

	if err := form.SetKnob("thickness", 99); err != nil {
		t.Fatal(err)
	}
	other, _ := state.Current().Game("other.exe")
	first, _ := state.Current().Game("stalzone.exe")
	if other.Crosshair.Thickness != 16 || first.Crosshair.Thickness == 16 {
		t.Errorf("thickness %d for other.exe and %d for stalzone.exe; want 16 for other.exe alone", other.Crosshair.Thickness, first.Crosshair.Thickness)
	}
	if err := form.SetKnob("size", 3); !errors.Is(err, crosshair.ErrUnknownKnob) {
		t.Errorf("got %v, want ErrUnknownKnob", err)
	}
}

func TestTheColoursOfTheCrosshairAndItsOutlineAreSet(t *testing.T) {
	form, state, _ := newForm(t, listed())

	red, blue := crosshair.RGB{R: 255}, crosshair.RGB{B: 255}
	if err := errors.Join(form.SetColor(red), form.SetOutlineColor(blue)); err != nil {
		t.Fatal(err)
	}
	if game, _ := state.Current().Game("stalzone.exe"); game.Crosshair.Color != red || game.Crosshair.Outline.Color != blue {
		t.Errorf("got %+v and %+v", game.Crosshair.Color, game.Crosshair.Outline.Color)
	}
}

func TestAPictureIsAddedShownAndScaledInItsRange(t *testing.T) {
	form, state, _ := newForm(t, listed())

	if err := form.AddPicture(pngOf(t, 9)); err != nil {
		t.Fatal(err)
	}
	shown := form.Screen().Game
	if !shown.Picture || shown.Crosshair.Image == "" {
		t.Fatalf("got %+v, want the picture on show", shown)
	}
	if shown.Scale.Label != form.Word("scale") || shown.Scale.Knob != crosshair.PictureScale() || shown.Scale.Value != 100 {
		t.Errorf("the scale reads %+v", shown.Scale)
	}

	for _, c := range []struct{ asked, want int }{{500, 400}, {10, 25}, {150, 150}} {
		form.SetScale(c.asked)
		if game, _ := state.Current().Game("stalzone.exe"); game.Crosshair.Scale != c.want {
			t.Errorf("scale %d holds %d, want %d", c.asked, game.Crosshair.Scale, c.want)
		}
	}
}

func TestABrokenPictureIsRefusedInWordsAndChangesNothing(t *testing.T) {
	form, state, _ := newForm(t, listed())
	before := state.Current()

	if err := form.AddPicture([]byte("not a png")); !errors.Is(err, crosshair.ErrNotPNG) {
		t.Errorf("got %v, want ErrNotPNG", err)
	}
	if got := form.Screen().Problem; got != form.Word("error.not-png") {
		t.Errorf("the problem reads %q", got)
	}
	if !equal(before, state.Current()) {
		t.Error("the settings changed")
	}
}

func TestThePictureTabStaysOpenWithoutAPictureUntilAnotherGameIsChosen(t *testing.T) {
	form, _, _ := newForm(t, listed())

	form.ShowPicture()
	if shown := form.Screen().Game; !shown.Picture || shown.Crosshair.Image != "" {
		t.Errorf("got %+v, want the empty picture tab", shown)
	}
	form.Choose("stalzone.exe")
	if !form.Screen().Game.Picture {
		t.Error("choosing the game on show closed its picture tab")
	}

	form.Choose("other.exe")
	form.Choose("stalzone.exe")
	if form.Screen().Game.Picture {
		t.Error("the picture tab is still open after another game was shown")
	}
}

func TestGoingBackToTheReadyMadeDropsThePictureAndKeepsTheShape(t *testing.T) {
	form, state, _ := newForm(t, listed())
	form.AddPicture(pngOf(t, 9))

	if err := form.ShowReady(); err != nil {
		t.Fatal(err)
	}
	game, _ := state.Current().Game("stalzone.exe")
	if form.Screen().Game.Picture || game.Crosshair.Image != "" || game.Crosshair.Shape != crosshair.Dot {
		t.Errorf("got %+v, want the dot back without a picture", game.Crosshair)
	}
}

func TestAGameThatNamesAPictureOpensOnThePictureTab(t *testing.T) {
	pictured := listed()
	pictured.Games[1].Crosshair.Image = strings.Repeat("ab", 32) + ".png"
	form, _, _ := newForm(t, pictured)

	form.Choose("other.exe")
	if !form.Screen().Game.Picture {
		t.Error("a game that names a picture opens on the ready-made shapes")
	}
}

func TestTheOffsetIsSetNudgedByTheStepAndCentredForTheShownGame(t *testing.T) {
	form, state, _ := newForm(t, listed())
	form.Choose("other.exe")

	offsetOf := func(exe string) profiles.Offset {
		game, _ := state.Current().Game(exe)
		return game.Offset
	}

	form.SetOffset(5, -7)
	if got := offsetOf("other.exe"); got != (profiles.Offset{X: 5, Y: -7}) {
		t.Fatalf("got %+v after setting it", got)
	}
	form.Nudge(1, 0)
	if got := offsetOf("other.exe"); got != (profiles.Offset{X: 6, Y: -7}) {
		t.Errorf("got %+v after a nudge by one", got)
	}
	form.SetFarStep(true)
	form.Nudge(0, -1)
	if got := offsetOf("other.exe"); got != (profiles.Offset{X: 6, Y: -17}) || !form.Screen().FarStep {
		t.Errorf("got %+v after a nudge by ten", got)
	}
	form.Centre()
	if got := offsetOf("other.exe"); got != (profiles.Offset{}) || offsetOf("stalzone.exe") != (profiles.Offset{}) {
		t.Errorf("got %+v after centring", got)
	}
	if form.Screen().Game.Offset != offsetOf("other.exe") {
		t.Error("the screen shows another offset than the settings hold")
	}
}

func TestTheOffsetStaysWithin500PixelsOfTheCentre(t *testing.T) {
	form, state, _ := newForm(t, listed())

	form.SetOffset(900, -900)
	form.SetFarStep(true)
	form.Nudge(1, -1)
	if game, _ := state.Current().Game("stalzone.exe"); game.Offset != (profiles.Offset{X: 500, Y: -500}) {
		t.Errorf("got %+v", game.Offset)
	}
}

func TestATypedOffsetIsReadAsAWholeNumber(t *testing.T) {
	cases := map[string]struct {
		want int
		ok   bool
	}{
		"1.6":   {want: 2, ok: true},
		" -3 ":  {want: -3, ok: true},
		"-2.5":  {want: -3, ok: true},
		"1e3":   {want: 1000, ok: true},
		"1e300": {want: 1000000, ok: true},
		"":      {ok: false},
		"abc":   {ok: false},
		"NaN":   {ok: false},
		"Inf":   {ok: false},
	}

	for text, c := range cases {
		if got, ok := settings.Whole(text); ok != c.ok || ok && got != c.want {
			t.Errorf("%q reads %d, %v; want %d, %v", text, got, ok, c.want, c.ok)
		}
	}
}

func TestACrosshairIsSavedUnderANameUsedAndForgotten(t *testing.T) {
	form, state, _ := newForm(t, listed())
	shownBefore, _ := state.Current().Game("stalzone.exe")

	if err := form.Save("  Мой  "); err != nil {
		t.Fatal(err)
	}
	saved := state.Current().Saved
	if len(saved) != 1 || saved[0].Name != "Мой" || saved[0].Crosshair != shownBefore.Crosshair {
		t.Fatalf("saved %+v, want the shown crosshair under its trimmed name", saved)
	}

	form.SetShape(crosshair.Circle)
	if err := form.UseSaved(0); err != nil {
		t.Fatal(err)
	}
	if game, _ := state.Current().Game("stalzone.exe"); game.Crosshair != shownBefore.Crosshair {
		t.Errorf("got %+v back, want the saved one", game.Crosshair)
	}

	if err := form.ForgetSaved(0); err != nil || len(state.Current().Saved) != 0 {
		t.Errorf("got %v with %d saved, want none", err, len(state.Current().Saved))
	}
	if err := form.UseSaved(0); !errors.Is(err, profiles.ErrNoSaved) || form.Screen().Problem != form.Word("error.no-saved") {
		t.Errorf("got %v reading %q, want ErrNoSaved in words", err, form.Screen().Problem)
	}
}

func TestABlankNameSavesNothing(t *testing.T) {
	form, state, _ := newForm(t, listed())

	if err := form.Save("   "); err != nil || len(state.Current().Saved) != 0 {
		t.Errorf("got %v with %+v saved", err, state.Current().Saved)
	}
}

func TestASavedCrosshairBringsItsOwnTab(t *testing.T) {
	s := listed()
	pictured := profiles.DefaultCrosshair()
	pictured.Image = strings.Repeat("cd", 32) + ".png"
	s.Saved = []profiles.Saved{{Name: "Картинка", Crosshair: pictured}, {Name: "Готовый", Crosshair: profiles.DefaultCrosshair()}}
	form, _, _ := newForm(t, s)

	form.UseSaved(0)
	if !form.Screen().Game.Picture {
		t.Error("a saved picture opens on the ready-made shapes")
	}
	form.ShowPicture()
	form.UseSaved(1)
	if form.Screen().Game.Picture {
		t.Error("a saved ready-made crosshair leaves the picture tab open")
	}
}

func TestTheHotkeysAreListedInWords(t *testing.T) {
	form, _, _ := newForm(t, listed())

	rows := form.Screen().Hotkeys
	var fields []string
	for _, row := range rows {
		fields = append(fields, row.Field)
		if row.Label != form.Word("hotkey."+row.Field) || row.Label == "" || row.Text != hotkeys.Get(profiles.Defaults().Hotkeys, row.Field) || row.Taken != "" || row.Suggestion != "" {
			t.Errorf("row %+v", row)
		}
	}
	if !slices.Equal(fields, hotkeys.Fields()) {
		t.Errorf("fields %v, want %v", fields, hotkeys.Fields())
	}
}

func TestATakenHotkeyIsNamedWithAFreeOneToTake(t *testing.T) {
	form, state, m := newForm(t, listed())
	m.taken = []string{"Ctrl+Shift+C", "Ctrl+Shift+Right", "Ctrl+Shift+Left"}

	rows := form.Screen().Hotkeys
	if next := rows[4]; next.Taken != "Ctrl+Shift+C" || next.Suggestion != "Ctrl+Shift+Alt+C" {
		t.Errorf("next reads %+v", next)
	}
	if nudge := rows[1]; nudge.Taken != "Ctrl+Shift+Left" || nudge.Suggestion == "" {
		t.Errorf("nudge reads %+v", nudge)
	}

	if err := form.TakeSuggestion("next"); err != nil {
		t.Fatal(err)
	}
	if got := state.Current().Hotkeys.Next; got != "Ctrl+Shift+Alt+C" {
		t.Errorf("next holds %q after taking the offer", got)
	}
	if got := form.Screen().Hotkeys[4].Taken; got != "" {
		t.Errorf("next still reads taken by %q", got)
	}
}

func TestAHotkeyIsCheckedBeforeItIsKept(t *testing.T) {
	form, state, _ := newForm(t, listed())

	if err := form.SetHotkey("toggle", "Shift+X"); !errors.Is(err, hotkeys.ErrBadHotkey) || form.Screen().Problem != form.Word("error.bad-hotkey") {
		t.Errorf("got %v reading %q", err, form.Screen().Problem)
	}
	if err := form.SetHotkey("toggle", "Ctrl+Shift+C"); !errors.Is(err, hotkeys.ErrSameHotkey) {
		t.Errorf("got %v, want ErrSameHotkey", err)
	}
	if err := form.SetHotkey("jump", "Ctrl+J"); !errors.Is(err, hotkeys.ErrNoField) {
		t.Errorf("got %v, want ErrNoField", err)
	}
	if got := state.Current().Hotkeys.Toggle; got != "Ctrl+Shift+X" {
		t.Errorf("toggle holds %q after the refusals", got)
	}

	if err := form.SetHotkey("toggle", "Ctrl+Alt+F9"); err != nil || state.Current().Hotkeys.Toggle != "Ctrl+Alt+F9" {
		t.Errorf("got %v holding %q", err, state.Current().Hotkeys.Toggle)
	}
}

func pictureWithBlock(t *testing.T, side, block int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	from := (side - block) / 2
	for y := from; y < from+block; y++ {
		for x := from; x < from+block; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	var file bytes.Buffer
	if err := png.Encode(&file, img); err != nil {
		t.Fatal(err)
	}

	return file.Bytes()
}

func rowRun(img *image.RGBA, y int, want color.RGBA) int {
	run := 0
	for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
		if img.RGBAAt(x, y) == want {
			run++
		}
	}

	return run
}

func TestThePreviewShowsTheCrosshairOnADarkAndALightSquare(t *testing.T) {
	form, state, _ := newForm(t, listed())

	dark, light, err := form.Preview(160)
	if err != nil {
		t.Fatal(err)
	}
	if !dark.Rect.Eq(image.Rect(0, 0, 160, 160)) || !light.Rect.Eq(dark.Rect) {
		t.Fatalf("sizes %v and %v, want 160 by 160", dark.Rect, light.Rect)
	}
	if dark.RGBAAt(0, 0) != (color.RGBA{0x22, 0x22, 0x22, 0xFF}) || light.RGBAAt(0, 0) != (color.RGBA{0xDD, 0xDD, 0xDD, 0xFF}) {
		t.Errorf("corners %v and %v, want the dark and the light backdrop", dark.RGBAAt(0, 0), light.RGBAAt(0, 0))
	}

	game, _ := state.Current().Game("stalzone.exe")
	green := color.RGBA{game.Crosshair.Color.R, game.Crosshair.Color.G, game.Crosshair.Color.B, 0xFF}
	if dark.RGBAAt(80, 80) != green || light.RGBAAt(80, 80) != green {
		t.Errorf("centres %v and %v, want the crosshair's own colour", dark.RGBAAt(80, 80), light.RGBAAt(80, 80))
	}

	form.SetColor(crosshair.RGB{R: 255})
	if dark, _, _ := form.Preview(160); dark.RGBAAt(80, 80) != (color.RGBA{0xFF, 0, 0, 0xFF}) {
		t.Errorf("after a change of colour the centre is %v, want red", dark.RGBAAt(80, 80))
	}
}

func TestASmallCrosshairIsEnlargedByWholePixels(t *testing.T) {
	form, state, _ := newForm(t, listed())
	game, _ := state.Current().Game("stalzone.exe")
	natural, _ := profiles.Store{Disk: &disk{files: map[string][]byte{}}, Dir: "NormPrichel"}.Paint(game.Crosshair)
	green := color.RGBA{game.Crosshair.Color.R, game.Crosshair.Color.G, game.Crosshair.Color.B, 0xFF}
	dot := rowRun(natural, natural.Rect.Dy()/2, green)

	for side, zoom := range map[int]int{160: 4, 16: 3, 12: 2, 8: 1, 5: 1} {
		dark, _, _ := form.Preview(side)
		if got := rowRun(dark, side/2, green); got != dot*zoom {
			t.Errorf("in %d pixels the dot is %d wide, want %d times %d", side, got, zoom, dot)
		}
	}
}

func TestALargePictureIsShrunkToFitTheSquare(t *testing.T) {
	form, _, _ := newForm(t, listed())
	if err := form.AddPicture(pictureWithBlock(t, 400, 40)); err != nil {
		t.Fatal(err)
	}

	dark, _, err := form.Preview(160)
	if err != nil {
		t.Fatal(err)
	}
	red := color.RGBA{0xFF, 0, 0, 0xFF}
	if got := rowRun(dark, 80, red); got < 14 || got > 18 {
		t.Errorf("the 40 pixel block shrunk into 160 is %d wide, want about 16", got)
	}
}

func TestAMissingPictureLeavesBareSquaresAndIsWordedBelowARefusal(t *testing.T) {
	file := pngOf(t, 9)
	sum := sha256.Sum256(file)
	missing := listed()
	missing.Games[0].Crosshair.Image = hex.EncodeToString(sum[:]) + ".png"
	form, _, _ := newForm(t, missing)

	dark, _, err := form.Preview(40)
	if !errors.Is(err, profiles.ErrNoPicture) || dark.RGBAAt(20, 20) != (color.RGBA{0x22, 0x22, 0x22, 0xFF}) {
		t.Errorf("got %v with the centre %v, want ErrNoPicture on a bare square", err, dark.RGBAAt(20, 20))
	}
	if got := form.Screen().Problem; got != form.Word("error.no-picture") {
		t.Errorf("the problem reads %q", got)
	}

	form.SetHotkey("toggle", "Shift+X")
	if got := form.Screen().Problem; got != form.Word("error.bad-hotkey") {
		t.Errorf("a refusal reads %q, want it over the missing picture", got)
	}
	form.Rename("S")
	if got := form.Screen().Problem; got != form.Word("error.no-picture") {
		t.Errorf("after the refusal cleared the problem reads %q, want the missing picture again", got)
	}

	if err := form.AddPicture(file); err != nil {
		t.Fatal(err)
	}
	if got := form.Screen().Problem; got != "" {
		t.Errorf("the picture is back, yet the problem reads %q", got)
	}
}

func TestWithNoGamesThePreviewIsBare(t *testing.T) {
	form, _, _ := newForm(t, profiles.Defaults())

	dark, light, err := form.Preview(20)
	if err != nil || dark.RGBAAt(10, 10) != (color.RGBA{0x22, 0x22, 0x22, 0xFF}) || light.RGBAAt(10, 10) != (color.RGBA{0xDD, 0xDD, 0xDD, 0xFF}) {
		t.Errorf("got %v with centres %v and %v", err, dark.RGBAAt(10, 10), light.RGBAAt(10, 10))
	}
}

func TestResettingTheCrosshairBringsBackTheDefaultAndLeavesTheRestOfTheGame(t *testing.T) {
	s := listed()
	s.Games[0].Offset, s.Games[0].HideWhileAiming = profiles.Offset{X: 4, Y: -2}, true
	s.Games[0].Crosshair.Color = crosshair.RGB{R: 255}
	form, state, _ := newForm(t, s)
	form.ShowPicture()

	if err := form.ResetCrosshair(); err != nil {
		t.Fatal(err)
	}

	game, other := state.Current().Games[0], state.Current().Games[1]
	if game.Crosshair != profiles.DefaultCrosshair() {
		t.Errorf("the crosshair is %+v after the reset, want the default", game.Crosshair)
	}
	if game.Offset != (profiles.Offset{X: 4, Y: -2}) || !game.HideWhileAiming || game.Name != "STALZONE" || !game.Enabled {
		t.Errorf("the reset touched the rest of the game: %+v", game)
	}
	if other.Crosshair != s.Games[1].Crosshair {
		t.Errorf("the reset reached the other game: %+v", other.Crosshair)
	}
	if form.Screen().Game.Picture {
		t.Error("the picture tab stays open over the ready-made crosshair the reset brought back")
	}
}
