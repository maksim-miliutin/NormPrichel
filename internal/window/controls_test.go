package window_test

import (
	"cmp"
	"slices"
	"strconv"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
	"github.com/maksim-miliutin/NormPrichel/internal/settings"
	"github.com/maksim-miliutin/NormPrichel/internal/window"
)

// The words the tabs are kept under, in the order their buttons stand.
var tabWords = []string{"crosshair", "hotkeys", "general"}

func words(key string) string {
	return settings.Word("ru", key)
}

func gameScreen(shape crosshair.Shape) settings.Screen {
	c := profiles.DefaultCrosshair()
	c.Shape = shape
	view := &settings.GameView{
		Exe:       "stalzone.exe",
		Name:      "STALZONE",
		Enabled:   true,
		Crosshair: c,
		Scale:     settings.KnobView{Knob: crosshair.PictureScale(), Label: words("scale"), Value: 150},
		Offset:    profiles.Offset{X: 3, Y: -7},
	}
	for _, knob := range crosshair.Knobs(shape) {
		view.Knobs = append(view.Knobs, settings.KnobView{Knob: knob, Label: words(knob.Name), Value: c.Value(knob.Name)})
	}

	return settings.Screen{
		Visible: true,
		Games: []settings.GameRow{
			{Exe: "stalzone.exe", Name: "STALZONE", Enabled: true, Chosen: true},
			{Exe: "other.exe", Name: "Other"},
		},
		Game:    view,
		Hotkeys: freeHotkeys(),
		Version: "v1.2.3",
	}
}

func freeHotkeys() []settings.HotkeyRow {
	var rows []settings.HotkeyRow
	for i, field := range hotkeys.Fields() {
		rows = append(rows, settings.HotkeyRow{Field: field, Label: words("hotkey." + field), Text: "Ctrl+Alt+F" + strconv.Itoa(i+1)})
	}

	return rows
}

func on(b bool) *bool {
	return &b
}

func everyScreen() map[string]settings.Screen {
	empty := settings.Screen{Hotkeys: freeHotkeys(), Autostart: on(false)}
	picture := gameScreen(crosshair.Cross)
	picture.Game.Picture = true
	keeping := gameScreen(crosshair.CrossDot)
	keeping.Saved = []string{"Мой", "Второй"}
	keeping.Problem = "Это не PNG."
	keeping.Hotkeys[0].Taken, keeping.Hotkeys[0].Suggestion = "Ctrl+Alt+F1", "Ctrl+Alt+F9"
	keeping.Hotkeys[2].Taken = "Ctrl+Alt+F3"

	screens := map[string]settings.Screen{}
	for name, screen := range map[string]settings.Screen{"no games": empty, "a ready-made crosshair": gameScreen(crosshair.Dot), "a picture": picture, "saved crosshairs and a problem": keeping} {
		screens[name] = screen
		for _, tab := range tabWords[1:] {
			screen.Tab = tab
			screens[name+" on "+tab] = screen
		}
	}

	return screens
}

func overlap(a, b window.Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

func shownOnly(controls []window.Control) []window.Control {
	return slices.DeleteFunc(slices.Clone(controls), func(c window.Control) bool { return !c.Shown })
}

func withWord(t *testing.T, controls []window.Control, word string) window.Control {
	t.Helper()
	i := slices.IndexFunc(controls, func(c window.Control) bool { return c.Word == word })
	if i < 0 {
		t.Fatalf("no control shows %q", word)
	}

	return controls[i]
}

func withText(t *testing.T, controls []window.Control, text string) window.Control {
	t.Helper()
	i := slices.IndexFunc(controls, func(c window.Control) bool { return c.Text == text })
	if i < 0 {
		t.Fatalf("no control reads %q", text)
	}

	return controls[i]
}

func ofKind(controls []window.Control, kind window.Kind) []window.Control {
	return slices.DeleteFunc(slices.Clone(controls), func(c window.Control) bool { return c.Kind != kind })
}

func TestShownControlsNeverOverlapWhateverTheScreen(t *testing.T) {
	for name, screen := range everyScreen() {
		shown := shownOnly(window.Controls(screen, words))
		for i, a := range shown {
			for _, b := range shown[i+1:] {
				if overlap(a.Rect, b.Rect) {
					t.Errorf("%s: %q at %+v overlaps %q at %+v", name, a.Text, a.Rect, b.Text, b.Rect)
				}
			}
		}
	}
}

func TestWithoutGamesTheListAndTheHintShowAndTheGameDoesNot(t *testing.T) {
	own := map[string][]string{
		"crosshair": {"noGames"},
		"hotkeys":   {"hotkeysHint"},
		"general":   {"risk", "resetAll"},
	}
	for _, tab := range tabWords {
		screen := everyScreen()["no games"]
		screen.Tab = tab
		controls := window.Controls(screen, words)

		for _, word := range append([]string{"games", "addGame"}, own[tab]...) {
			if !withWord(t, controls, word).Shown {
				t.Errorf("%s: %q is hidden", tab, word)
			}
		}
		for _, word := range []string{"remove", "resetCrosshair", "copyCode", "pasteCode", "preview", "position", "centre", "hideWhileAiming", "saved", "saveCurrent", "color", "upload"} {
			if withWord(t, controls, word).Shown {
				t.Errorf("%s: %q shows with no game to apply it to", tab, word)
			}
		}
		languages := func(c window.Control) bool { return c.Text == "Русский" || c.Text == "English" }
		for _, kind := range []window.Kind{window.Preview, window.Slider, window.Colour, window.Choice} {
			if shown := slices.DeleteFunc(shownOnly(ofKind(controls, kind)), languages); len(shown) != 0 {
				t.Errorf("%s: %d controls of kind %s show with no game", tab, len(shown), kind)
			}
		}
	}
}

func TestOnlyTheOpenTabShowsAndItsButtonIsDown(t *testing.T) {
	holds := map[string][]string{
		"crosshair": {"readyMade", "preview", "copyCode", "saved", "position", "centre"},
		"hotkeys":   {"hotkeysHint"},
		"general":   {"autostart", "language", "risk", "resetAll"},
	}
	for _, tab := range append([]string{""}, tabWords...) {
		screen := gameScreen(crosshair.Dot)
		screen.Tab, screen.Autostart = tab, on(false)
		controls := window.Controls(screen, words)
		open := cmp.Or(tab, "crosshair")

		for _, word := range tabWords {
			if button := withWord(t, controls, word); !button.Shown || button.On != (word == open) {
				t.Errorf("open %q: the %s button shows %v and is down %v", tab, word, button.Shown, button.On)
			}
		}
		for owner, held := range holds {
			for _, word := range held {
				if shown := withWord(t, controls, word).Shown; shown != (owner == open) {
					t.Errorf("open %q: %q of the %s tab shows %v", tab, word, owner, shown)
				}
			}
		}
		for _, word := range []string{"visible", "games", "addGame", "name", "enabled", "hideWhileAiming", "remove"} {
			if !withWord(t, controls, word).Shown {
				t.Errorf("open %q: %q is hidden, want it beside every tab", tab, word)
			}
		}
	}
}

func TestTheKnobsOfTheShapeFillTheSlotsInOrder(t *testing.T) {
	for _, shape := range crosshair.Shapes() {
		screen := gameScreen(shape)
		controls := window.Controls(screen, words)

		var got []string
		for _, c := range shownOnly(ofKind(controls, window.Slider)) {
			got = append(got, c.Knob.Name)
			if c.Knob.Value != screen.Game.Crosshair.Value(c.Knob.Name) || c.Knob.Max == 0 {
				t.Errorf("%s: the %s slider holds %+v", shape, c.Knob.Name, c.Knob)
			}
		}
		var want []string
		for _, knob := range crosshair.Knobs(shape) {
			want = append(want, knob.Name)
			label := withText(t, controls, words(knob.Name))
			value := strconv.Itoa(screen.Game.Crosshair.Value(knob.Name))
			if i := slices.IndexFunc(controls, func(c window.Control) bool { return c.ID == label.ID+1 }); controls[i].Text != value {
				t.Errorf("%s: %s reads %q beside its label, want %q", shape, knob.Name, controls[i].Text, value)
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: the sliders show %v, want %v", shape, got, want)
		}
	}
}

func TestAPictureShowsItsScaleInsteadOfTheShapes(t *testing.T) {
	controls := window.Controls(everyScreen()["a picture"], words)

	sliders := shownOnly(ofKind(controls, window.Slider))
	if len(sliders) != 1 || sliders[0].Knob.Name != "scale" || sliders[0].Knob.Value != 150 {
		t.Fatalf("the picture shows sliders %+v, want the scale alone at 150", sliders)
	}
	upload := withWord(t, controls, "upload")
	if !withText(t, controls, "150").Shown || !upload.Shown {
		t.Error("the scale's value or the upload button is hidden")
	}
	if gap := sliders[0].Rect.Y - (upload.Rect.Y + upload.Rect.H); gap < 0 || gap > 12 {
		t.Errorf("the scale sits %d pixels below the upload button", gap)
	}
	if withText(t, controls, words("shape.cross")).Shown || withWord(t, controls, "color").Shown {
		t.Error("the shapes still show beside the picture")
	}
	if !withWord(t, controls, "picture").On || withWord(t, controls, "readyMade").On {
		t.Error("the picture tab is not the one down")
	}
}

func TestEveryShapeHasAButtonThatDrawsIt(t *testing.T) {
	controls := window.Controls(gameScreen(crosshair.Dot), words)

	var drawn []crosshair.Shape
	for _, c := range shownOnly(ofKind(controls, window.Choice)) {
		if c.Shape == "" {
			continue
		}
		drawn = append(drawn, c.Shape)
		if c.Text != words("shape."+string(c.Shape)) {
			t.Errorf("the %s button is named %q", c.Shape, c.Text)
		}
	}
	if !slices.Equal(drawn, crosshair.Shapes()) {
		t.Errorf("buttons draw %v, want every shape %v", drawn, crosshair.Shapes())
	}
}

func TestTheSwatchesOfferReadyColoursAndMarkTheOneInUse(t *testing.T) {
	screen := gameScreen(crosshair.Dot)
	controls := window.Controls(screen, words)
	swatches, squares := ofKind(controls, window.Swatch), ofKind(controls, window.Colour)
	if len(swatches) != 10 {
		t.Fatalf("%d swatches, want eight colours for the crosshair and two for its outline", len(swatches))
	}

	seen := map[crosshair.RGB]bool{}
	for i, c := range swatches[:8] {
		if seen[c.Colour] || c.On != (i == 0) {
			t.Errorf("swatch %d of %v: repeated %v, down %v", i, c.Colour, seen[c.Colour], c.On)
		}
		seen[c.Colour] = true
	}
	if swatches[0].Colour != crosshair.Default().Color || squares[0].On {
		t.Errorf("the first swatch is %v and the own colour is down %v, want the default green down alone", swatches[0].Colour, squares[0].On)
	}
	white := crosshair.RGB{R: 255, G: 255, B: 255}
	if edging := swatches[8:]; edging[0].Colour != (crosshair.RGB{}) || edging[1].Colour != white || !edging[0].On || edging[1].On || squares[1].On {
		t.Errorf("outline swatches %v and %v, down %v and %v", edging[0].Colour, edging[1].Colour, edging[0].On, edging[1].On)
	}

	screen.Game.Crosshair.Color = crosshair.RGB{R: 1, G: 2, B: 3}
	screen.Game.Crosshair.Outline.Color = crosshair.RGB{R: 255}
	controls = window.Controls(screen, words)
	for _, c := range ofKind(controls, window.Swatch) {
		if c.On {
			t.Errorf("the swatch of %v is down for a colour of the player's own", c.Colour)
		}
	}
	for i, own := range ofKind(controls, window.Colour) {
		if !own.On {
			t.Errorf("own colour square %d is up while its colour is the player's own", i)
		}
	}
}

func TestTheChosenShapeTabAndStepAreTheOnesDown(t *testing.T) {
	screen := gameScreen(crosshair.Circle)
	screen.FarStep = true
	var down []string
	for _, c := range shownOnly(ofKind(window.Controls(screen, words), window.Choice)) {
		if c.On {
			down = append(down, c.Text)
		}
	}

	if want := []string{words("readyMade"), words("shape.circle"), words("outline.line"), "10 px"}; !slices.Equal(down, want) {
		t.Errorf("down %q, want %q", down, want)
	}
}

func TestATakenHotkeyNamesTheCombinationAndOffersAFreeOne(t *testing.T) {
	controls := window.Controls(everyScreen()["saved crosshairs and a problem on hotkeys"], words)
	fields := ofKind(controls, window.Hotkey)

	if !fields[0].On || fields[1].On || !fields[2].On {
		t.Errorf("fields marked taken: %v %v %v, want the first and the third", fields[0].On, fields[1].On, fields[2].On)
	}
	if take := withText(t, controls, words("take")+" Ctrl+Alt+F9"); !take.Shown || take.ID != fields[0].ID+1 {
		t.Errorf("the offer to take Ctrl+Alt+F9 is %+v", take)
	}
	if note := withText(t, controls, "Ctrl+Alt+F1 "+words("taken")); !note.Shown {
		t.Error("the first field does not say who holds Ctrl+Alt+F1")
	}

	var offers, notes int
	for _, c := range shownOnly(controls) {
		switch {
		case c.Kind == window.Button && c.ID == fields[2].ID+1:
			t.Errorf("an offer shows with nothing free to take: %q", c.Text)
		case c.Kind == window.Button && c.ID > fields[0].ID && c.ID < fields[4].ID+3:
			offers++
		case c.Kind == window.Hint && c.ID > fields[0].ID && c.ID < fields[4].ID+3:
			notes++
		}
	}
	if offers != 1 || notes != 2 {
		t.Errorf("%d offers and %d notes show, want 1 and 2", offers, notes)
	}
}

func column(r window.Rect) int {
	columns := 0
	for _, edge := range []int{232, 684} {
		if r.X >= edge {
			columns++
		}
	}

	return columns
}

func TestTabGoesDownEachColumn(t *testing.T) {
	for name, screen := range everyScreen() {
		last := map[int]window.Rect{}
		for _, c := range shownOnly(window.Controls(screen, words)) {
			if c.Kind == window.Label || c.Kind == window.Heading || c.Kind == window.Hint || c.Kind == window.Problem || c.Kind == window.Preview || c.Kind == window.Divider || c.Rect.Y < 40 {
				continue
			}
			before, seen := last[column(c.Rect)]
			if seen && (c.Rect.Y < before.Y || c.Rect.Y == before.Y && c.Rect.X < before.X) {
				t.Errorf("%s: Tab goes from %+v up or back to %q at %+v", name, before, c.Text, c.Rect)
			}
			last[column(c.Rect)] = c.Rect
		}
	}
}

func TestTheGamesListMarksTheChosenAndTheOffGames(t *testing.T) {
	screen := gameScreen(crosshair.Dot)
	screen.Games[0].Chosen, screen.Games[1].Chosen = false, true
	list := ofKind(window.Controls(screen, words), window.List)[0]

	if want := []string{"STALZONE", "Other · " + words("gameOff")}; !slices.Equal(list.Rows, want) || list.Row != 1 {
		t.Errorf("rows %q with row %d chosen, want %q with the second", list.Rows, list.Row, want)
	}
}

func TestTheSavedListLeavesItsRowToThePlayer(t *testing.T) {
	list := ofKind(window.Controls(everyScreen()["saved crosshairs and a problem"], words), window.List)[1]

	if !slices.Equal(list.Rows, []string{"Мой", "Второй"}) || list.Row != -1 || !list.Shown {
		t.Errorf("the saved list is %+v", list)
	}
}

func TestTheAutostartSwitchShowsWhatWindowsSays(t *testing.T) {
	for _, c := range []struct {
		starts       *bool
		shown, ticks bool
	}{
		{starts: nil, shown: false, ticks: false},
		{starts: on(false), shown: true, ticks: false},
		{starts: on(true), shown: true, ticks: true},
	} {
		screen := gameScreen(crosshair.Dot)
		screen.Autostart, screen.Tab = c.starts, "general"
		if got := withWord(t, window.Controls(screen, words), "autostart"); got.Shown != c.shown || got.On != c.ticks {
			t.Errorf("with %v the switch shows %v and ticks %v", c.starts, got.Shown, got.On)
		}
	}
}

func TestEveryShownButtonAndHeadingSaysWhatItIs(t *testing.T) {
	for name, screen := range everyScreen() {
		for _, c := range shownOnly(window.Controls(screen, words)) {
			worded := c.Kind == window.Button || c.Kind == window.Choice || c.Kind == window.Switch || c.Kind == window.Tab || c.Kind == window.Heading || c.Kind == window.Label
			if worded && c.Text == "" {
				t.Errorf("%s: control %d of kind %s shows no words", name, c.ID, c.Kind)
			}
		}
	}
}

func TestTheSwitchesAndFieldsShowTheSettings(t *testing.T) {
	screen := gameScreen(crosshair.Dot)
	screen.Game.Enabled = false
	screen.Game.HideWhileAiming = true
	controls := window.Controls(screen, words)

	if c := withWord(t, controls, "visible"); !c.On || !c.Shown {
		t.Errorf("visible: %+v", c)
	}
	if withWord(t, controls, "enabled").On {
		t.Error("the game shows as on")
	}
	for _, text := range []string{"STALZONE", "stalzone.exe"} {
		if !withText(t, controls, text).Shown {
			t.Errorf("%q is hidden", text)
		}
	}
	if c := withWord(t, controls, "hideWhileAiming"); !c.On || !c.Shown {
		t.Errorf("hiding while aiming: %+v, want shown and on", c)
	}
	squares := ofKind(controls, window.Colour)
	if squares[0].Colour != screen.Game.Crosshair.Color || squares[1].Colour != screen.Game.Crosshair.Outline.Color {
		t.Errorf("the colour squares show %v and %v, want %v and %v", squares[0].Colour, squares[1].Colour, screen.Game.Crosshair.Color, screen.Game.Crosshair.Outline.Color)
	}

	for _, text := range []string{"3", "-7"} {
		if !withText(t, controls, text).Shown {
			t.Errorf("%q is hidden", text)
		}
	}

	screen.Tab = "general"
	if version := words("version") + " v1.2.3"; !withText(t, window.Controls(screen, words), version).Shown {
		t.Errorf("%q is hidden", version)
	}
}

func TestTheLanguageButtonsNameEachLanguageInItselfAndMarkTheOneSpoken(t *testing.T) {
	screen := gameScreen(crosshair.Dot)
	screen.Language, screen.Tab = "en", "general"
	controls := window.Controls(screen, words)

	russian, english := withText(t, controls, "Русский"), withText(t, controls, "English")
	if russian.On || !english.On || !russian.Shown || !english.Shown {
		t.Errorf("Русский %+v and English %+v, want both shown and English down", russian, english)
	}
	if !withText(t, window.Controls(everyScreen()["no games on general"], words), "Русский").Shown {
		t.Error("the language cannot be chosen before a game is added")
	}
}

func TestTheProblemShowsOnlyWhileThereIsOne(t *testing.T) {
	for name, screen := range everyScreen() {
		c := ofKind(window.Controls(screen, words), window.Problem)[0]
		if c.Shown != (screen.Problem != "") || c.Text != screen.Problem {
			t.Errorf("%s: the problem line is %+v", name, c)
		}
	}
}

func TestOnlyTheNameToSaveUnderIsThePlayersOwn(t *testing.T) {
	controls := window.Controls(gameScreen(crosshair.Dot), words)

	var typed []int
	for _, c := range controls {
		if c.Typed {
			typed = append(typed, c.ID)
		}
	}
	if save := withWord(t, controls, "saveCurrent"); len(typed) != 1 || typed[0] != save.ID-1 {
		t.Errorf("the player's own fields are %v, want the one before the save button", typed)
	}
}

func TestTheCornerAndOutlineChoicesShowTheCrosshairAndHideWhereTheyDoNothing(t *testing.T) {
	screen := gameScreen(crosshair.Cross)
	screen.Game.Crosshair.Corners, screen.Game.Crosshair.Outline.Shadow = crosshair.Round, true
	controls := window.Controls(screen, words)

	for word, on := range map[string]bool{"corners.sharp": false, "corners.soft": false, "corners.round": true, "outline.line": false, "outline.shadow": true} {
		if c := withWord(t, controls, word); !c.Shown || c.On != on {
			t.Errorf("%s shows %v and is down %v, want shown and down %v", word, c.Shown, c.On, on)
		}
	}

	controls = window.Controls(gameScreen(crosshair.Circle), words)
	for _, word := range []string{"corners", "corners.sharp", "corners.round"} {
		if withWord(t, controls, word).Shown {
			t.Errorf("%s shows for a circle, which has no corners", word)
		}
	}
	if !withWord(t, controls, "outline.shadow").Shown {
		t.Error("the shadow hides for a circle, which has an outline")
	}
}

func TestTheShapeButtonsDrawWithTheCornersOfTheCrosshair(t *testing.T) {
	screen := gameScreen(crosshair.Cross)
	screen.Game.Crosshair.Corners = crosshair.Soft

	for _, c := range window.Controls(screen, words) {
		if c.Shape != "" && c.Corners != crosshair.Soft {
			t.Errorf("the %s button draws with %q corners, want soft", c.Shape, c.Corners)
		}
	}
}
