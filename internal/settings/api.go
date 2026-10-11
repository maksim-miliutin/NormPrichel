package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"slices"
	"strconv"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/live"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var ErrBadRequest = errors.New("settings: the request does not read")

const (
	maxBody    = 64 << 10 // bytes of JSON in one change
	maxPicture = 4 << 20  // bytes read of an upload; the picture rules refuse past 2 MB anyway
)

type State interface {
	Current() profiles.Settings
	Last() profiles.Crosshair
	Change(edit func(profiles.Settings) (profiles.Settings, error)) (profiles.Settings, error)
}

type Pictures interface {
	AddPicture(file []byte) (string, error)
	Paint(c profiles.Crosshair) (*image.RGBA, error)
}

type Page struct {
	State    State
	Pictures Pictures
	Windows  func() ([]foreground.Choice, error)
	Taken    func(bindings []hotkeys.Binding) []hotkeys.Binding
	Version  string
	Language string
}

type edit = func(profiles.Settings) (profiles.Settings, error)

type view struct {
	Settings profiles.Settings `json:"settings"`
	Shapes   []crosshair.Shape `json:"shapes"`
	Taken    []takenKey        `json:"taken"`
	Version  string            `json:"version"`
	Language string            `json:"language"`
}

type takenKey struct {
	Field      string `json:"field"`
	Hotkey     string `json:"hotkey"`
	Suggestion string `json:"suggestion,omitempty"`
}

// The page names a refusal by its code and words it in the player's language.
var reasons = []struct {
	err  error
	code string
}{
	{crosshair.ErrNotPNG, "not-png"},
	{crosshair.ErrFileTooBig, "too-big"},
	{crosshair.ErrTooLarge, "too-large"},
	{crosshair.ErrUnknownShape, "unknown-shape"},
	{profiles.ErrNoPicture, "no-picture"},
	{hotkeys.ErrBadHotkey, "bad-hotkey"},
	{hotkeys.ErrSameHotkey, "same-hotkey"},
	{profiles.ErrListed, "listed"},
	{profiles.ErrNoExe, "no-exe"},
	{profiles.ErrNoGame, "no-game"},
	{profiles.ErrNoSaved, "no-saved"},
	{live.ErrRefused, "refused"},
	{ErrBadRequest, "bad-request"},
}

func (p Page) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/state", p.state)
	mux.HandleFunc("GET /api/windows", p.windows)
	mux.HandleFunc("GET /api/preview", p.preview)
	mux.HandleFunc("POST /api/pictures", p.addPicture)
	mux.HandleFunc("PUT /api/visible", p.setVisible)
	mux.HandleFunc("PUT /api/hotkeys", p.setHotkeys)
	mux.HandleFunc("POST /api/games", p.addGame)
	mux.HandleFunc("PATCH /api/games/{exe}", p.editGame)
	mux.HandleFunc("DELETE /api/games/{exe}", p.removeGame)
	mux.HandleFunc("PUT /api/games/{exe}/crosshair", p.setCrosshair)
	mux.HandleFunc("PUT /api/games/{exe}/offset", p.setOffset)
	mux.HandleFunc("POST /api/games/{exe}/nudge", p.nudge)
	mux.HandleFunc("POST /api/saved", p.save)
	mux.HandleFunc("DELETE /api/saved/{index}", p.forget)

	return mux
}

func (p Page) state(w http.ResponseWriter, _ *http.Request) {
	current := p.State.Current()
	respond(w, view{
		Settings: current,
		Shapes:   crosshair.Shapes(),
		Taken:    p.taken(current.Hotkeys),
		Version:  p.Version,
		Language: p.Language,
	})
}

// Each field is named once, by the first of its hotkeys that another program holds.
func (p Page) taken(keys profiles.Hotkeys) []takenKey {
	found := []takenKey{}
	bindings, _ := hotkeys.Bindings(keys)
	for _, b := range p.Taken(bindings) {
		if slices.ContainsFunc(found, func(t takenKey) bool { return t.Field == b.Field }) {
			continue
		}
		suggestion, _ := hotkeys.Suggest(keys, b.Field, p.Taken)
		found = append(found, takenKey{Field: b.Field, Hotkey: b.Text, Suggestion: suggestion})
	}

	return found
}

func (p Page) windows(w http.ResponseWriter, _ *http.Request) {
	choices, err := p.Windows()
	if err != nil {
		failed(w, err)
		return
	}

	respond(w, choices)
}

func (p Page) preview(w http.ResponseWriter, r *http.Request) {
	var c profiles.Crosshair
	if err := json.Unmarshal([]byte(r.URL.Query().Get("crosshair")), &c); err != nil {
		failed(w, fmt.Errorf("%w: %w", ErrBadRequest, err))
		return
	}

	img, err := p.Pictures.Paint(c)
	if err != nil {
		failed(w, err)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, img)
}

func (p Page) addPicture(w http.ResponseWriter, r *http.Request) {
	file, err := io.ReadAll(io.LimitReader(r.Body, maxPicture))
	if err != nil {
		failed(w, fmt.Errorf("%w: %w", ErrBadRequest, err))
		return
	}

	name, err := p.Pictures.AddPicture(file)
	if err != nil {
		failed(w, err)
		return
	}

	respond(w, map[string]string{"image": name})
}

func (p Page) setVisible(w http.ResponseWriter, r *http.Request) {
	on, ok := read[bool](w, r)
	if !ok {
		return
	}

	p.apply(w, r, func(s profiles.Settings) (profiles.Settings, error) {
		s.Visible = on
		return s, nil
	})
}

func (p Page) setHotkeys(w http.ResponseWriter, r *http.Request) {
	keys, ok := read[profiles.Hotkeys](w, r)
	if !ok {
		return
	}

	p.apply(w, r, func(s profiles.Settings) (profiles.Settings, error) {
		if _, err := hotkeys.Bindings(keys); err != nil {
			return s, err
		}
		s.Hotkeys = keys
		return s, nil
	})
}

func (p Page) addGame(w http.ResponseWriter, r *http.Request) {
	chosen, ok := read[struct {
		Exe  string `json:"exe"`
		Name string `json:"name"`
	}](w, r)
	if !ok {
		return
	}

	last := p.State.Last()
	p.apply(w, r, func(s profiles.Settings) (profiles.Settings, error) {
		return s.WithGame(profiles.Game{Exe: chosen.Exe, Name: chosen.Name, Enabled: true, Crosshair: last})
	})
}

func (p Page) editGame(w http.ResponseWriter, r *http.Request) {
	change, ok := read[struct {
		Name    *string `json:"name"`
		Enabled *bool   `json:"enabled"`
	}](w, r)
	if !ok {
		return
	}

	p.edit(w, r, func(g profiles.Game) profiles.Game {
		if change.Name != nil {
			g.Name = *change.Name
		}
		if change.Enabled != nil {
			g.Enabled = *change.Enabled
		}
		return g
	})
}

func (p Page) removeGame(w http.ResponseWriter, r *http.Request) {
	p.apply(w, r, func(s profiles.Settings) (profiles.Settings, error) {
		return s.WithoutGame(r.PathValue("exe"))
	})
}

func (p Page) setCrosshair(w http.ResponseWriter, r *http.Request) {
	c, ok := read[profiles.Crosshair](w, r)
	if !ok {
		return
	}

	p.edit(w, r, func(g profiles.Game) profiles.Game {
		g.Crosshair = c
		return g
	})
}

func (p Page) setOffset(w http.ResponseWriter, r *http.Request) {
	offset, ok := read[profiles.Offset](w, r)
	if !ok {
		return
	}

	p.edit(w, r, func(g profiles.Game) profiles.Game {
		g.Offset = offset
		return g
	})
}

func (p Page) nudge(w http.ResponseWriter, r *http.Request) {
	step, ok := read[profiles.Offset](w, r)
	if !ok {
		return
	}

	p.edit(w, r, func(g profiles.Game) profiles.Game {
		g.Offset = profiles.Offset{X: g.Offset.X + step.X, Y: g.Offset.Y + step.Y}
		return g
	})
}

func (p Page) save(w http.ResponseWriter, r *http.Request) {
	entry, ok := read[struct {
		Name      string             `json:"name"`
		Crosshair profiles.Crosshair `json:"crosshair"`
	}](w, r)
	if !ok {
		return
	}

	p.apply(w, r, func(s profiles.Settings) (profiles.Settings, error) {
		return s.WithSaved(profiles.Saved{Name: entry.Name, Crosshair: entry.Crosshair}), nil
	})
}

func (p Page) forget(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		failed(w, fmt.Errorf("%w: %w", ErrBadRequest, err))
		return
	}

	p.apply(w, r, func(s profiles.Settings) (profiles.Settings, error) {
		return s.WithoutSaved(index)
	})
}

func (p Page) edit(w http.ResponseWriter, r *http.Request, change func(profiles.Game) profiles.Game) {
	p.apply(w, r, func(s profiles.Settings) (profiles.Settings, error) {
		return s.Edited(r.PathValue("exe"), change)
	})
}

// Every change answers with the whole state, so the page always shows what is in force.
func (p Page) apply(w http.ResponseWriter, r *http.Request, change edit) {
	if _, err := p.State.Change(change); err != nil {
		failed(w, err)
		return
	}

	p.state(w, r)
}

func read[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var body T
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		failed(w, fmt.Errorf("%w: %w", ErrBadRequest, err))
		return body, false
	}

	return body, true
}

func respond(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(value)
}

func failed(w http.ResponseWriter, err error) {
	code, status := "failed", http.StatusInternalServerError
	for _, reason := range reasons {
		if errors.Is(err, reason.err) {
			code, status = reason.code, http.StatusBadRequest
			break
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": code, "detail": err.Error()})
}
