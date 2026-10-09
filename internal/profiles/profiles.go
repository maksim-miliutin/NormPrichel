package profiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

var ErrBroken = errors.New("profiles: the settings file is broken")

const version = 1

const maxOffset = 500

var pictureName = regexp.MustCompile(`^[0-9a-f]{64}\.png$`)

type Settings struct {
	Version int     `json:"version"`
	Visible bool    `json:"visible"`
	Hotkeys Hotkeys `json:"hotkeys"`
	Games   []Game  `json:"games"`
	Saved   []Saved `json:"saved"`
}

type Hotkeys struct {
	Toggle   string `json:"toggle"`
	Nudge    string `json:"nudge"`
	NudgeFar string `json:"nudgeFar"`
	Center   string `json:"center"`
	Next     string `json:"next"`
}

type Game struct {
	Exe       string    `json:"exe"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	Crosshair Crosshair `json:"crosshair"`
	Offset    Offset    `json:"offset"`
}

type Offset struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// A crosshair is drawn from its style unless it names a picture; the style is kept either way,
// so switching back finds it as it was.
type Crosshair struct {
	crosshair.Style
	Image string `json:"image,omitempty"`
	Scale int    `json:"scale,omitempty"` // percent
}

type Saved struct {
	Name string `json:"name"`
	Crosshair
}

func Defaults() Settings {
	return Settings{
		Version: version,
		Visible: true,
		Hotkeys: Hotkeys{
			Toggle:   "Ctrl+Shift+X",
			Nudge:    "Ctrl+Shift",
			NudgeFar: "Ctrl+Shift+Alt",
			Center:   "Ctrl+Shift+Home",
			Next:     "Ctrl+Shift+C",
		},
		Games: []Game{},
		Saved: []Saved{},
	}
}

func Parse(text []byte) (Settings, error) {
	settings := Defaults()
	if err := json.Unmarshal(text, &settings); err != nil {
		return Settings{}, fmt.Errorf("%w: %w", ErrBroken, err)
	}

	if settings.Version != version {
		return Settings{}, fmt.Errorf("%w: version %d", ErrBroken, settings.Version)
	}

	kept := Settings{Games: []Game{}}
	for _, game := range settings.Games {
		if _, listed := kept.Game(game.Exe); listed || game.Exe == "" {
			continue
		}
		if err := game.Crosshair.check(); err != nil {
			return Settings{}, err
		}

		game.Offset = game.Offset.clamped()
		kept.Games = append(kept.Games, game)
	}

	saved := []Saved{}
	for _, entry := range settings.Saved {
		if err := entry.check(); err != nil {
			return Settings{}, err
		}
		saved = append(saved, entry)
	}

	return Settings{Version: settings.Version, Visible: settings.Visible, Hotkeys: settings.Hotkeys, Games: kept.Games, Saved: saved}, nil
}

func (s Settings) Encode() ([]byte, error) {
	return json.MarshalIndent(s, "", "\t")
}

func (s Settings) Game(exe string) (Game, bool) {
	for _, game := range s.Games {
		if strings.EqualFold(game.Exe, exe) {
			return game, true
		}
	}

	return Game{}, false
}

func (g *Game) UnmarshalJSON(text []byte) error {
	type plain Game
	decoded := plain{Enabled: true, Crosshair: defaultCrosshair()}
	if err := json.Unmarshal(text, &decoded); err != nil {
		return err
	}

	*g = Game(decoded)

	return nil
}

// Saved embeds Crosshair, so a decoder on Crosshair would be promoted and swallow Name.
func (s *Saved) UnmarshalJSON(text []byte) error {
	type plain Saved
	decoded := plain{Crosshair: defaultCrosshair()}
	if err := json.Unmarshal(text, &decoded); err != nil {
		return err
	}

	*s = Saved(decoded)

	return nil
}

func defaultCrosshair() Crosshair {
	return Crosshair{Style: crosshair.Default(), Scale: 100}
}

func (c Crosshair) check() error {
	if !slices.Contains(crosshair.Shapes(), c.Shape) {
		return fmt.Errorf("%w: shape %q", ErrBroken, c.Shape)
	}
	if c.Image != "" && !pictureName.MatchString(c.Image) {
		return fmt.Errorf("%w: picture %q", ErrBroken, c.Image)
	}

	return nil
}

func (o Offset) clamped() Offset {
	return Offset{X: min(max(o.X, -maxOffset), maxOffset), Y: min(max(o.Y, -maxOffset), maxOffset)}
}
