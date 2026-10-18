package window_test

import (
	"bytes"
	"io"
	"slices"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
	"github.com/maksim-miliutin/NormPrichel/internal/settings"
	"github.com/maksim-miliutin/NormPrichel/internal/window"
)

func first(t *testing.T, s profiles.Settings) profiles.Game {
	t.Helper()
	if len(s.Games) == 0 {
		t.Fatal("no games are listed")
	}

	return s.Games[0]
}

func TestClickingASwitchFlipsWhatItShows(t *testing.T) {
	a, state, m, _ := newActions(t, listed())

	for _, word := range []string{"visible", "enabled", "autostart", "hideWhileAiming"} {
		if err := a.Click(control(t, a, worded(word))); err != nil {
			t.Fatal(err)
		}
	}

	s := state.Current()
	if s.Visible || first(t, s).Enabled || !m.starts || !first(t, s).HideWhileAiming {
		t.Errorf("visible %v, enabled %v, autostart %v, hiding while aiming %v: want off, off, on, on",
			s.Visible, first(t, s).Enabled, m.starts, first(t, s).HideWhileAiming)
	}
}

func TestTheLanguageButtonsSwitchTheWordsOfTheWindow(t *testing.T) {
	a, state, _, _ := newActions(t, listed())

	if err := a.Click(control(t, a, reading("English"))); err != nil {
		t.Fatal(err)
	}
	if got := state.Current().Language; got != "en" {
		t.Errorf("after English the settings keep %q", got)
	}
	control(t, a, reading(settings.Word("en", "addGame")))

	if err := a.Click(control(t, a, reading("Русский"))); err != nil {
		t.Fatal(err)
	}
	if got := state.Current().Language; got != "ru" {
		t.Errorf("after Русский the settings keep %q", got)
	}
	control(t, a, reading(settings.Word("ru", "addGame")))
}

func TestTheTabButtonsOpenTheirTabs(t *testing.T) {
	a, _, _, _ := newActions(t, listed())

	for _, word := range []string{"general", "hotkeys", "position", "crosshair"} {
		if err := a.Click(control(t, a, worded(word))); err != nil {
			t.Fatal(err)
		}
		if got := a.Form.Screen().Tab; got != word {
			t.Errorf("after the %s button the open tab is %q", word, got)
		}
	}
}

func TestAGameIsRemovedOnlyOnceThePlayerAgrees(t *testing.T) {
	a, state, _, p := newActions(t, listed())
	remove := control(t, a, worded("remove"))

	a.Click(remove)
	if len(state.Current().Games) != 2 {
		t.Fatal("the game went without the player agreeing")
	}
	p.agrees = true
	a.Click(remove)

	if games := state.Current().Games; len(games) != 1 || games[0].Exe != "other.exe" {
		t.Errorf("games left: %+v, want other.exe alone", games)
	}
	if want := a.Form.Word("confirm"); len(p.asked) != 2 || p.asked[0] != want {
		t.Errorf("asked %q, want %q twice", p.asked, want)
	}
}

func TestTheCrosshairIsResetOnlyOnceThePlayerAgrees(t *testing.T) {
	a, state, _, p := newActions(t, listed())
	a.Form.SetShape(crosshair.Circle)
	reset := control(t, a, worded("resetCrosshair"))

	a.Click(reset)
	if first(t, state.Current()).Crosshair.Shape != crosshair.Circle {
		t.Fatal("the crosshair went back to the default without the player agreeing")
	}
	p.agrees = true
	a.Click(reset)

	if got := first(t, state.Current()).Crosshair; got != profiles.DefaultCrosshair() {
		t.Errorf("the crosshair is %+v after agreeing, want the default", got)
	}
	if want := a.Form.Word("confirmReset"); !slices.Equal(p.asked, []string{want, want}) {
		t.Errorf("asked %q, want %q twice", p.asked, want)
	}
}

func TestEverythingIsResetOnlyOnceThePlayerAgrees(t *testing.T) {
	a, state, _, p := newActions(t, listed())
	reset := control(t, a, worded("resetAll"))

	a.Click(reset)
	if len(state.Current().Games) != 2 {
		t.Fatal("the settings went back to a first start without the player agreeing")
	}
	p.agrees = true
	a.Click(reset)

	if got := state.Current(); len(got.Games) != 0 {
		t.Errorf("games left after agreeing: %+v", got.Games)
	}
	if want := a.Form.Word("confirmResetAll"); !slices.Equal(p.asked, []string{want, want}) {
		t.Errorf("asked %q, want %q twice", p.asked, want)
	}
}

func TestTheCodeButtonsGoThroughTheClipboard(t *testing.T) {
	a, state, _, p := newActions(t, listed())
	copyCode, pasteCode := control(t, a, worded("copyCode")), control(t, a, worded("pasteCode"))

	a.Click(copyCode)
	if want := first(t, state.Current()).Crosshair.Code(); !slices.Equal(p.copied, []string{want}) {
		t.Errorf("the clipboard got %q, want %q", p.copied, want)
	}
	if button := withID(window.Controls(a.Form.Screen(), a.Form.Word), copyCode); !button.On || button.Text != a.Form.Word("copied") {
		t.Errorf("after the copy the button reads %q, on %v; want it to say the code is copied", button.Text, button.On)
	}

	pasted := crosshair.Default()
	pasted.Shape = crosshair.Triangle
	p.clipboard = pasted.Code()
	a.Click(pasteCode)
	if got := first(t, state.Current()).Crosshair.Style; got != pasted {
		t.Errorf("after the paste the crosshair is %+v, want the triangle", got)
	}
}

func withID(controls []window.Control, id int) window.Control {
	for _, c := range controls {
		if c.ID == id {
			return c
		}
	}

	return window.Control{}
}

func TestAGameIsAddedFromTheWindowThePlayerPicks(t *testing.T) {
	a, state, m, p := newActions(t, profiles.Defaults())
	m.windows = []foreground.Choice{{Exe: "notepad.exe", Title: "Блокнот"}, {Exe: "stalzone.exe", Title: "STALZONE"}}
	add := control(t, a, worded("addGame"))

	a.Click(add)
	if len(state.Current().Games) != 0 {
		t.Fatal("a game was added though the player picked none")
	}
	p.pick, p.picked = 1, true
	a.Click(add)

	if want := []string{"Блокнот — notepad.exe", "STALZONE — stalzone.exe"}; !slices.Equal(p.offered, want) || p.none != a.Form.Word("noWindows") {
		t.Errorf("offered %q with %q for none, want %q", p.offered, p.none, want)
	}
	if games := state.Current().Games; len(games) != 1 || games[0].Exe != "stalzone.exe" {
		t.Errorf("games: %+v, want stalzone.exe", games)
	}
	if got := a.Form.Screen().Game; got == nil || got.Exe != "stalzone.exe" {
		t.Error("the added game is not the one on show")
	}
}

func TestAShapeButtonDrawsItsShapeAndTheTabsSwitchToThePicture(t *testing.T) {
	a, state, _, _ := newActions(t, listed())

	a.Click(control(t, a, reading(a.Form.Word("shape.circle"))))
	if got := first(t, state.Current()).Crosshair.Shape; got != crosshair.Circle {
		t.Fatalf("shape %s, want circle", got)
	}

	a.Click(control(t, a, worded("picture")))
	if !a.Form.Screen().Game.Picture {
		t.Fatal("the picture tab did not open")
	}
	a.Click(control(t, a, worded("readyMade")))
	if a.Form.Screen().Game.Picture {
		t.Error("the ready-made tab did not come back")
	}
}

func TestTheColourDialogStartsFromTheColourAndChangesItOnlyWhenChosen(t *testing.T) {
	a, state, _, p := newActions(t, listed())
	square := control(t, a, nth(window.Colour, 1))
	before := first(t, state.Current()).Crosshair

	p.colour = crosshair.RGB{R: 255}
	a.Click(square)
	if first(t, state.Current()).Crosshair != before || p.started != before.Outline.Color {
		t.Fatalf("a closed dialog changed the crosshair, or started from %v", p.started)
	}
	p.colourChosen = true
	a.Click(square)

	if got := first(t, state.Current()).Crosshair; got.Outline.Color != p.colour || got.Color != before.Color {
		t.Fatalf("outline %v and fill %v, want %v and %v", got.Outline.Color, got.Color, p.colour, before.Color)
	}

	p.colour = crosshair.RGB{B: 255}
	a.Click(control(t, a, nth(window.Colour, 0)))
	if got := first(t, state.Current()).Crosshair; got.Color != p.colour || got.Outline.Color != (crosshair.RGB{R: 255}) || p.started != before.Color {
		t.Errorf("fill %v and outline %v after the fill square, started from %v", got.Color, got.Outline.Color, p.started)
	}
}

func TestASwatchSetsItsColourInOneClick(t *testing.T) {
	a, state, _, _ := newActions(t, listed())
	white := crosshair.RGB{R: 255, G: 255, B: 255}
	red := control(t, a, func(c window.Control) bool { return c.Kind == window.Swatch && c.Colour == (crosshair.RGB{R: 255}) })
	edging := control(t, a, nth(window.Swatch, 9))

	a.Click(red)
	a.Click(edging)

	if got := first(t, state.Current()).Crosshair; got.Color != (crosshair.RGB{R: 255}) || got.Outline.Color != white {
		t.Errorf("colour %v and outline %v, want red with a white outline", got.Color, got.Outline.Color)
	}
}

func TestAPictureFileBecomesTheCrosshair(t *testing.T) {
	a, state, _, p := newActions(t, listed())
	p.picture = io.NopCloser(bytes.NewReader(pngOf(t, 16)))

	if err := a.Click(control(t, a, worded("upload"))); err != nil {
		t.Fatal(err)
	}

	if first(t, state.Current()).Crosshair.Image == "" || !a.Form.Screen().Game.Picture {
		t.Error("the picture is not the crosshair")
	}
}

func TestAPictureFileIsReadNoFurtherThanAnyPictureCouldGo(t *testing.T) {
	a, state, _, p := newActions(t, listed())
	endless := &counted{}
	p.picture = endless

	a.Click(control(t, a, worded("upload")))

	if endless.read > 4<<20+64<<10 || !endless.closed {
		t.Errorf("read %d bytes and closed %v", endless.read, endless.closed)
	}
	if first(t, state.Current()).Crosshair.Image != "" || a.Form.Screen().Problem != a.Form.Word("error.too-big") {
		t.Errorf("the endless file is the crosshair, or the problem reads %q", a.Form.Screen().Problem)
	}
}

func TestThePadMovesByTheChosenStepAndTheCentreButtonReturns(t *testing.T) {
	a, state, _, _ := newActions(t, listed())

	a.Click(control(t, a, reading("→")))
	a.Click(control(t, a, reading("↑")))
	a.Click(control(t, a, reading("10 px")))
	a.Click(control(t, a, reading("←")))
	a.Click(control(t, a, reading("↓")))
	if got := first(t, state.Current()).Offset; got != (profiles.Offset{X: -9, Y: 9}) {
		t.Fatalf("offset %+v, want -9, 9", got)
	}
	a.Click(control(t, a, reading("1 px")))
	a.Click(control(t, a, reading("→")))
	if got := first(t, state.Current()).Offset; got.X != -8 {
		t.Fatalf("x %d after one more step, want -8", got.X)
	}

	a.Click(control(t, a, worded("centre")))
	if got := first(t, state.Current()).Offset; got != (profiles.Offset{}) {
		t.Errorf("offset %+v after the centre button", got)
	}
}

func TestATypedOffsetAppliesWhenTheFieldIsLeft(t *testing.T) {
	a, state, _, p := newActions(t, listed())
	x := control(t, a, reading("X")) + 1
	y := control(t, a, reading("Y")) + 1

	for _, c := range []struct {
		id   int
		text string
		want profiles.Offset
	}{
		{x, "12", profiles.Offset{X: 12}},
		{y, " -7.6 ", profiles.Offset{X: 12, Y: -8}},
		{x, "двенадцать", profiles.Offset{X: 12, Y: -8}},
	} {
		p.texts[c.id] = c.text
		if err := a.Leave(c.id); err != nil {
			t.Fatal(err)
		}
		if got := first(t, state.Current()).Offset; got != c.want {
			t.Errorf("after %q the offset is %+v, want %+v", c.text, got, c.want)
		}
	}
}

func TestTheNameChangesOnLeavingAndTheSaveNameWaitsForItsButton(t *testing.T) {
	a, state, _, p := newActions(t, listed())
	name := control(t, a, reading("STALZONE"))
	saveName := control(t, a, worded("saveCurrent")) - 1

	p.texts[name] = "Сталзона"
	p.texts[saveName] = "  Мой  "
	a.Leave(name)
	a.Leave(saveName)
	if s := state.Current(); first(t, s).Name != "Сталзона" || len(s.Saved) != 0 {
		t.Fatalf("name %q and saved %+v: want the new name and nothing saved yet", first(t, s).Name, s.Saved)
	}

	a.Enter(saveName)
	if s := state.Current(); len(s.Saved) != 1 || s.Saved[0].Name != "Мой" {
		t.Errorf("saved %+v after Enter, want Мой", s.Saved)
	}
}

func TestASavedCrosshairIsUsedOrForgottenByItsRow(t *testing.T) {
	s := listed()
	saved := profiles.DefaultCrosshair()
	saved.Shape = crosshair.Tee
	s.Saved = []profiles.Saved{{Name: "Т", Crosshair: saved}}
	a, state, _, p := newActions(t, s)
	use, forget := control(t, a, worded("use")), control(t, a, worded("forget"))
	list := control(t, a, nth(window.List, 1))

	a.Click(use)
	if first(t, state.Current()).Crosshair.Shape == crosshair.Tee || a.Form.Screen().Problem != "" {
		t.Fatalf("with no row chosen a crosshair was used, or the problem reads %q", a.Form.Screen().Problem)
	}
	p.rows[list] = 0
	a.Open(list)
	if first(t, state.Current()).Crosshair.Shape != crosshair.Tee {
		t.Fatal("a double click did not use the saved crosshair")
	}

	a.Click(forget)
	if len(state.Current().Saved) != 1 {
		t.Fatal("the saved crosshair went without the player agreeing")
	}
	p.agrees = true
	a.Click(forget)
	if len(state.Current().Saved) != 0 || !slices.Equal(p.asked, []string{a.Form.Word("confirmForget"), a.Form.Word("confirmForget")}) {
		t.Errorf("saved %+v after agreeing, asked %q", state.Current().Saved, p.asked)
	}
}

func TestChoosingARowOfTheGamesShowsThatGame(t *testing.T) {
	a, _, _, p := newActions(t, listed())
	games := control(t, a, nth(window.List, 0))

	p.rows[games] = 2
	a.Select(games)
	p.rows[games] = 1
	a.Select(games)

	if got := a.Form.Screen().Game.Exe; got != "other.exe" {
		t.Errorf("on show: %s, want other.exe", got)
	}
}

func TestAFieldLeftAsItWasWritesNothing(t *testing.T) {
	a, state, _, p := newActions(t, listed())
	name := control(t, a, reading("STALZONE"))
	x := control(t, a, reading("X")) + 1
	toggle := control(t, a, nth(window.Hotkey, 0))
	gap := control(t, a, func(c window.Control) bool { return c.Kind == window.Slider && c.Knob.Name == "gap" })
	p.texts[name], p.texts[x], p.texts[toggle] = "STALZONE", "0", "Ctrl+Shift+X"

	for _, id := range []int{name, x, toggle} {
		a.Leave(id)
	}
	a.Slide(gap, first(t, state.Current()).Crosshair.Gap)

	if state.count != 0 {
		t.Errorf("%d changes were written", state.count)
	}
}

func TestOnlyAHotkeyFieldTakesPressesAndOnlyTheOfferTakesClicks(t *testing.T) {
	a, state, m, _ := newActions(t, listed())
	m.taken = []string{"Ctrl+Shift+X"}
	field := control(t, a, nth(window.Hotkey, 0))

	if taken, _ := a.Press(field+1, hotkeys.Control|hotkeys.Alt, 'K'); taken {
		t.Error("the offer button took a press")
	}
	a.Click(field)
	a.Click(field - 1)

	if state.count != 0 {
		t.Errorf("%d changes were written", state.count)
	}
}

func TestASliderSetsTheKnobBesideIt(t *testing.T) {
	a, state, _, _ := newActions(t, listed())
	gap := control(t, a, func(c window.Control) bool { return c.Kind == window.Slider && c.Knob.Name == "gap" })

	a.Slide(gap, 7)
	if got := first(t, state.Current()).Crosshair.Gap; got != 7 {
		t.Fatalf("gap %d, want 7", got)
	}

	a.Click(control(t, a, worded("picture")))
	a.Slide(control(t, a, func(c window.Control) bool { return c.Kind == window.Slider && c.Shown }), 175)
	if got := first(t, state.Current()).Crosshair.Scale; got != 175 {
		t.Errorf("scale %d, want 175", got)
	}
}

func TestAPressedCombinationBecomesTheHotkeyAndPlainKeysAreTyping(t *testing.T) {
	a, state, _, _ := newActions(t, listed())
	toggle := control(t, a, nth(window.Hotkey, 0))
	nudge := control(t, a, nth(window.Hotkey, 1))

	if taken, _ := a.Press(toggle, hotkeys.Shift, 'K'); taken {
		t.Error("Shift+K counts as a hotkey")
	}
	if taken, err := a.Press(toggle, hotkeys.Control|hotkeys.Alt, 'K'); !taken || err != nil {
		t.Fatalf("Ctrl+Alt+K: taken %v, %v", taken, err)
	}
	if taken, _ := a.Press(nudge, hotkeys.Control|hotkeys.Alt, 0x25); !taken {
		t.Error("Ctrl+Alt with an arrow is not taken for the nudge")
	}

	if keys := state.Current().Hotkeys; keys.Toggle != "Ctrl+Alt+K" || keys.Nudge != "Ctrl+Alt" {
		t.Errorf("toggle %q and nudge %q, want Ctrl+Alt+K and Ctrl+Alt", keys.Toggle, keys.Nudge)
	}
}

func TestATypedHotkeyAppliesWhenTheFieldIsLeft(t *testing.T) {
	a, state, _, p := newActions(t, listed())
	next := control(t, a, nth(window.Hotkey, 4))

	p.texts[next] = "Ctrl+Alt+N"
	a.Leave(next)

	if got := state.Current().Hotkeys.Next; got != "Ctrl+Alt+N" {
		t.Errorf("next %q, want Ctrl+Alt+N", got)
	}
}

func TestTheOfferedHotkeyIsTakenByItsButton(t *testing.T) {
	a, state, m, _ := newActions(t, listed())
	m.taken = []string{"Ctrl+Shift+X"}
	offer := a.Form.Screen().Hotkeys[0].Suggestion
	if offer == "" {
		t.Fatal("nothing is offered for a taken hotkey")
	}

	a.Click(control(t, a, reading(a.Form.Word("take")+" "+offer)))

	if got := state.Current().Hotkeys.Toggle; got != offer {
		t.Errorf("toggle %q, want the offered %q", got, offer)
	}
}

func TestTheGitHubButtonOpensTheRepository(t *testing.T) {
	a, _, _, p := newActions(t, listed())

	a.Click(control(t, a, reading("GitHub")))

	if !slices.Equal(p.visited, []string{"https://github.com/maksim-miliutin/NormPrichel"}) {
		t.Errorf("visited %q", p.visited)
	}
}
