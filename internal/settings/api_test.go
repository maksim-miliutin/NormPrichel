package settings_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
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

var errNoDesktop = errors.New("no desktop")

type site struct {
	page    http.Handler
	state   *live.State
	taken   []string
	windows []foreground.Choice
	fail    error

	starts      bool
	startsFault error
	switchFault error
}

func newSite(t *testing.T, s profiles.Settings) *site {
	t.Helper()

	st := &site{state: live.New(s, nowhere{}, func(error) {})}
	t.Cleanup(st.state.Close)
	st.page = settings.Page{
		State:        st.state,
		Pictures:     profiles.Store{Disk: &disk{files: map[string][]byte{}}, Dir: "NormPrichel"},
		Windows:      st.desktop,
		Taken:        st.held,
		Autostart:    st.autostart,
		SetAutostart: st.setAutostart,
		Version:      "v0.1.0",
		Language:     "ru",
	}.Handler()

	return st
}

func (st *site) desktop() ([]foreground.Choice, error) {
	if st.fail != nil {
		return nil, st.fail
	}

	return st.windows, nil
}

func (st *site) autostart() (bool, error) {
	return st.starts, st.startsFault
}

func (st *site) setAutostart(on bool) error {
	if st.switchFault != nil {
		return st.switchFault
	}
	st.starts = on

	return nil
}

func (st *site) held(bindings []hotkeys.Binding) []hotkeys.Binding {
	var taken []hotkeys.Binding
	for _, b := range bindings {
		if slices.Contains(st.taken, b.Text) {
			taken = append(taken, b)
		}
	}

	return taken
}

type view struct {
	Settings profiles.Settings `json:"settings"`
	Shapes   []string          `json:"shapes"`
	Taken    []struct {
		Field      string `json:"field"`
		Hotkey     string `json:"hotkey"`
		Suggestion string `json:"suggestion"`
	} `json:"taken"`
	Version   string `json:"version"`
	Language  string `json:"language"`
	Autostart *bool  `json:"autostart"`
}

type refusal struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}

func (st *site) call(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequest(method, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	st.page.ServeHTTP(w, r)

	return w
}

func (st *site) change(t *testing.T, method, path, body string) view {
	t.Helper()

	w := st.call(t, method, path, body)
	if w.Code != http.StatusOK {
		t.Fatalf("%s %s: status %d, %s", method, path, w.Code, w.Body)
	}
	var v view
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("%s %s: %v in %s", method, path, err, w.Body)
	}

	return v
}

func (st *site) refuse(t *testing.T, method, path, body, reason string) {
	t.Helper()

	before := st.state.Current()
	w := st.call(t, method, path, body)
	var got refusal
	json.Unmarshal(w.Body.Bytes(), &got)
	if w.Code != http.StatusBadRequest || got.Error != reason {
		t.Errorf("%s %s %s: status %d, %+v; want 400 and %q", method, path, body, w.Code, got, reason)
	}
	if after := st.state.Current(); !equal(before, after) {
		t.Errorf("%s %s %s: the settings changed although refused", method, path, body)
	}
}

func TestTheStateCarriesSettingsShapesVersionAndLanguage(t *testing.T) {
	st := newSite(t, listed())

	got := st.change(t, http.MethodGet, "/api/state", "")

	if !equal(got.Settings, listed()) {
		t.Errorf("settings %+v, want the ones in use", got.Settings)
	}
	if !slices.Equal(got.Shapes, []string{"dot", "cross", "cross-dot", "circle", "circle-dot", "t"}) {
		t.Errorf("shapes %v, want the six of the spec", got.Shapes)
	}
	if got.Version != "v0.1.0" || got.Language != "ru" || got.Taken == nil {
		t.Errorf("version %q, language %q, taken %v; want v0.1.0, ru and an empty list", got.Version, got.Language, got.Taken)
	}
}

func TestAGameAddedFromAWindowGetsTheCrosshairChosenLast(t *testing.T) {
	st := newSite(t, listed())

	got := st.change(t, http.MethodPost, "/api/games", `{"exe": "new.exe", "name": "New game"}`)

	game, ok := got.Settings.Game("new.exe")
	if !ok || game.Name != "New game" || !game.Enabled || game.Crosshair != crosshairOf(crosshair.Circle) {
		t.Errorf("got %+v, %v; want New game switched on with the circle of the last game", game, ok)
	}
	st.refuse(t, http.MethodPost, "/api/games", `{"exe": "NEW.EXE", "name": "again"}`, "listed")
	st.refuse(t, http.MethodPost, "/api/games", `{"exe": "", "name": "nameless"}`, "no-exe")
}

func TestAGameIsRenamedSwitchedOffAndRemoved(t *testing.T) {
	st := newSite(t, listed())

	renamed := st.change(t, http.MethodPatch, "/api/games/other.exe", `{"name": "Мой шутер"}`)
	if game, _ := renamed.Settings.Game("other.exe"); game.Name != "Мой шутер" || !game.Enabled {
		t.Errorf("after a rename: %+v, want the new name and the game still on", game)
	}

	paused := st.change(t, http.MethodPatch, "/api/games/OTHER.EXE", `{"enabled": false}`)
	if game, _ := paused.Settings.Game("other.exe"); game.Name != "Мой шутер" || game.Enabled {
		t.Errorf("after switching off: %+v, want the name kept and the game off", game)
	}

	removed := st.change(t, http.MethodDelete, "/api/games/other.exe", "")
	if _, ok := removed.Settings.Game("other.exe"); ok || len(removed.Settings.Games) != 1 {
		t.Errorf("after removal: games %+v, want stalzone.exe alone", removed.Settings.Games)
	}
	st.refuse(t, http.MethodDelete, "/api/games/other.exe", "", "no-game")
}

func TestACrosshairIsSetForItsGameAlone(t *testing.T) {
	st := newSite(t, listed())
	wanted := crosshairOf(crosshair.Tee)
	body, _ := json.Marshal(wanted)

	got := st.change(t, http.MethodPut, "/api/games/other.exe/crosshair", string(body))

	if other, _ := got.Settings.Game("other.exe"); other.Crosshair != wanted {
		t.Errorf("other.exe has %+v, want the tee", other.Crosshair)
	}
	if first, _ := got.Settings.Game("stalzone.exe"); first.Crosshair != crosshairOf(crosshair.Dot) {
		t.Errorf("stalzone.exe changed to %+v", first.Crosshair)
	}
	st.refuse(t, http.MethodPut, "/api/games/other.exe/crosshair", strings.Replace(string(body), `"t"`, `"star"`, 1), "refused")
}

func TestTheOffsetIsSetAndNudgedNoFurtherThan500(t *testing.T) {
	st := newSite(t, listed())

	set := st.change(t, http.MethodPut, "/api/games/other.exe/offset", `{"x": 900, "y": -12}`)
	if game, _ := set.Settings.Game("other.exe"); game.Offset != (profiles.Offset{X: 500, Y: -12}) {
		t.Errorf("after setting: %+v, want 500, -12", game.Offset)
	}

	nudged := st.change(t, http.MethodPost, "/api/games/other.exe/nudge", `{"x": -3, "y": 2}`)
	if game, _ := nudged.Settings.Game("other.exe"); game.Offset != (profiles.Offset{X: 497, Y: -10}) {
		t.Errorf("after a nudge: %+v, want 497, -10", game.Offset)
	}
	st.refuse(t, http.MethodPost, "/api/games/notepad.exe/nudge", `{"x": 1}`, "no-game")
}

func TestACrosshairIsSavedUnderANameAndForgottenByItsNumber(t *testing.T) {
	st := newSite(t, listed())
	body, _ := json.Marshal(map[string]any{"name": "Кольцо", "crosshair": crosshairOf(crosshair.Circle)})

	saved := st.change(t, http.MethodPost, "/api/saved", string(body))
	if len(saved.Settings.Saved) != 1 || saved.Settings.Saved[0].Name != "Кольцо" || saved.Settings.Saved[0].Crosshair != crosshairOf(crosshair.Circle) {
		t.Errorf("saved %+v, want the circle under Кольцо", saved.Settings.Saved)
	}

	st.refuse(t, http.MethodDelete, "/api/saved/1", "", "no-saved")
	st.refuse(t, http.MethodDelete, "/api/saved/first", "", "bad-request")
	forgotten := st.change(t, http.MethodDelete, "/api/saved/0", "")
	if len(forgotten.Settings.Saved) != 0 {
		t.Errorf("saved %+v, want nothing left", forgotten.Settings.Saved)
	}
}

func TestTheCrosshairIsHiddenAndShownEverywhere(t *testing.T) {
	st := newSite(t, listed())

	if got := st.change(t, http.MethodPut, "/api/visible", "false"); got.Settings.Visible {
		t.Error("the crosshair is still visible")
	}
	if got := st.change(t, http.MethodPut, "/api/visible", "true"); !got.Settings.Visible {
		t.Error("the crosshair is still hidden")
	}
}

func TestHotkeysAreCheckedBeforeTheyAreKept(t *testing.T) {
	st := newSite(t, listed())
	keys := profiles.Defaults().Hotkeys

	keys.Toggle = "Ctrl+Alt+F9"
	good, _ := json.Marshal(keys)
	if got := st.change(t, http.MethodPut, "/api/hotkeys", string(good)); got.Settings.Hotkeys != keys {
		t.Errorf("hotkeys %+v, want %+v", got.Settings.Hotkeys, keys)
	}

	keys.Toggle = "Shift+X"
	typing, _ := json.Marshal(keys)
	st.refuse(t, http.MethodPut, "/api/hotkeys", string(typing), "bad-hotkey")

	keys.Toggle = "Ctrl+Shift+C"
	twice, _ := json.Marshal(keys)
	st.refuse(t, http.MethodPut, "/api/hotkeys", string(twice), "same-hotkey")
}

func TestATakenHotkeyIsNamedOncePerFieldWithAFreeOneOffered(t *testing.T) {
	st := newSite(t, listed())
	st.taken = []string{"Ctrl+Shift+C", "Ctrl+Shift+Up", "Ctrl+Shift+Down"}

	got := st.change(t, http.MethodGet, "/api/state", "")

	want := map[string][2]string{
		"next":  {"Ctrl+Shift+C", "Ctrl+Shift+Alt+C"},
		"nudge": {"Ctrl+Shift+Up", "Shift+Alt"},
	}
	if len(got.Taken) != len(want) {
		t.Fatalf("taken %+v, want one entry for next and one for nudge", got.Taken)
	}
	for _, taken := range got.Taken {
		if w := want[taken.Field]; taken.Hotkey != w[0] || taken.Suggestion != w[1] {
			t.Errorf("%s: %s with %q offered, want %s with %q", taken.Field, taken.Hotkey, taken.Suggestion, w[0], w[1])
		}
	}
}

func TestAPictureIsTakenOnlyAsASmallPNG(t *testing.T) {
	st := newSite(t, listed())

	w := st.call(t, http.MethodPost, "/api/pictures", string(pngOf(t, 16)))
	var added struct {
		Image string `json:"image"`
	}
	json.Unmarshal(w.Body.Bytes(), &added)
	if w.Code != http.StatusOK || len(added.Image) != 68 {
		t.Errorf("status %d, %s; want 200 and the picture's name", w.Code, w.Body)
	}

	st.refuse(t, http.MethodPost, "/api/pictures", "not a picture", "not-png")
	st.refuse(t, http.MethodPost, "/api/pictures", string(pngOf(t, 600)), "too-large")
	st.refuse(t, http.MethodPost, "/api/pictures", strings.Repeat("x", 3<<20), "too-big")
}

func TestThePreviewIsTheCrosshairAsTheOverlayDrawsIt(t *testing.T) {
	st := newSite(t, listed())
	c := profiles.DefaultCrosshair()
	text, _ := json.Marshal(c)

	w := st.call(t, http.MethodGet, "/api/preview?crosshair="+url.QueryEscape(string(text)), "")
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("status %d with %q, want a PNG", w.Code, w.Header().Get("Content-Type"))
	}

	decoded, err := png.Decode(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	got := image.NewRGBA(decoded.Bounds())
	draw.Draw(got, got.Rect, decoded, decoded.Bounds().Min, draw.Src)
	want, _ := crosshair.Draw(c.Style)
	if !bytes.Equal(got.Pix, want.Pix) || got.Rect != want.Rect {
		t.Errorf("the preview is %v and differs from the drawn crosshair %v", got.Rect, want.Rect)
	}

	st.refuse(t, http.MethodGet, "/api/preview?crosshair=%7B", "", "bad-request")
	missing := strings.Repeat("ab", 32) + ".png"
	st.refuse(t, http.MethodGet, "/api/preview?crosshair="+url.QueryEscape(`{"shape": "dot", "image": "`+missing+`", "scale": 100}`), "", "no-picture")
}

func TestTheWindowsToAddComeFromTheDesktop(t *testing.T) {
	st := newSite(t, listed())
	st.windows = []foreground.Choice{{Title: "STALZONE", Exe: "stalzone.exe"}}

	w := st.call(t, http.MethodGet, "/api/windows", "")
	var got []foreground.Choice
	json.Unmarshal(w.Body.Bytes(), &got)
	if w.Code != http.StatusOK || !slices.Equal(got, st.windows) {
		t.Errorf("status %d, %+v; want the desktop's windows", w.Code, got)
	}

	st.fail = errNoDesktop
	if w := st.call(t, http.MethodGet, "/api/windows", ""); w.Code != http.StatusInternalServerError {
		t.Errorf("a failing desktop answers %d, want 500", w.Code)
	}
}

func TestABodyThatDoesNotReadIsRefused(t *testing.T) {
	st := newSite(t, listed())

	st.refuse(t, http.MethodPut, "/api/games/other.exe/offset", `{"x": 1,`, "bad-request")
	st.refuse(t, http.MethodPut, "/api/games/other.exe/offset", `{"x": 1, "z": 2}`, "bad-request")
	st.refuse(t, http.MethodPut, "/api/visible", `"yes"`, "bad-request")
}

func TestStartingWithWindowsIsShownAndSwitchedFromThePage(t *testing.T) {
	st := newSite(t, listed())

	if got := st.change(t, http.MethodGet, "/api/state", ""); got.Autostart == nil || *got.Autostart {
		t.Fatalf("got %v, want off", got.Autostart)
	}
	if got := st.change(t, http.MethodPut, "/api/autostart", "true"); got.Autostart == nil || !*got.Autostart || !st.starts {
		t.Errorf("got %v with Windows told %v; want on", got.Autostart, st.starts)
	}
	if got := st.change(t, http.MethodPut, "/api/autostart", "false"); got.Autostart == nil || *got.Autostart || st.starts {
		t.Errorf("got %v with Windows told %v; want off", got.Autostart, st.starts)
	}
}

func TestStartingWithWindowsThatCannotBeReadIsLeftUnsaid(t *testing.T) {
	st := newSite(t, listed())
	st.startsFault = errors.New("the registry is locked")

	if got := st.change(t, http.MethodGet, "/api/state", ""); got.Autostart != nil {
		t.Errorf("got %v, want nothing said", *got.Autostart)
	}
}

func TestStartingWithWindowsThatCannotBeSwitchedIsNamed(t *testing.T) {
	st := newSite(t, listed())
	st.switchFault = errors.New("access denied")

	st.refuse(t, http.MethodPut, "/api/autostart", "true", "autostart")
	st.refuse(t, http.MethodPut, "/api/autostart", `"yes"`, "bad-request")
}
