package hotkeys

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var (
	ErrBadHotkey  = errors.New("hotkeys: a hotkey is Ctrl, Alt or Win, maybe more modifiers, and one key")
	ErrSameHotkey = errors.New("hotkeys: two actions share one hotkey")
	ErrNoAction   = errors.New("hotkeys: no action under that number")
)

// The modifier values RegisterHotKey takes; NoRepeat keeps a held key from acting again.
const (
	Alt      uint32 = 0x0001
	Control  uint32 = 0x0002
	Shift    uint32 = 0x0004
	Win      uint32 = 0x0008
	NoRepeat uint32 = 0x4000
)

type Binding struct {
	Action    int
	Field     string // the hotkeys field of the settings file it comes from
	Modifiers uint32
	Key       uint32 // a Windows virtual-key code
	Repeat    bool   // a held key keeps acting
	Text      string
}

type field struct {
	name   string
	access func(h *profiles.Hotkeys) *string
	arrows bool
}

// The nudge fields hold modifiers alone, and each of them binds the four arrows.
var (
	toggleField = field{name: "toggle", access: func(h *profiles.Hotkeys) *string { return &h.Toggle }}
	nudgeField  = field{name: "nudge", access: func(h *profiles.Hotkeys) *string { return &h.Nudge }, arrows: true}
	farField    = field{name: "nudgeFar", access: func(h *profiles.Hotkeys) *string { return &h.NudgeFar }, arrows: true}
	centreField = field{name: "center", access: func(h *profiles.Hotkeys) *string { return &h.Center }}
	nextField   = field{name: "next", access: func(h *profiles.Hotkeys) *string { return &h.Next }}

	fields = []field{toggleField, nudgeField, farField, centreField, nextField}
)

type action struct {
	name  string
	field field
	arrow string
	apply func(s profiles.Settings, exe string) (profiles.Settings, error)
}

var actions = []action{
	{name: "toggle", field: toggleField, apply: toggled},
	{name: "nudge left", field: nudgeField, arrow: "Left", apply: nudged(-1, 0)},
	{name: "nudge right", field: nudgeField, arrow: "Right", apply: nudged(1, 0)},
	{name: "nudge up", field: nudgeField, arrow: "Up", apply: nudged(0, -1)},
	{name: "nudge down", field: nudgeField, arrow: "Down", apply: nudged(0, 1)},
	{name: "far nudge left", field: farField, arrow: "Left", apply: nudged(-10, 0)},
	{name: "far nudge right", field: farField, arrow: "Right", apply: nudged(10, 0)},
	{name: "far nudge up", field: farField, arrow: "Up", apply: nudged(0, -10)},
	{name: "far nudge down", field: farField, arrow: "Down", apply: nudged(0, 10)},
	{name: "centre", field: centreField, apply: centred},
	{name: "next crosshair", field: nextField, apply: cycled},
}

var modifiers = []struct {
	name  string
	alias string
	flag  uint32
}{
	{name: "Ctrl", alias: "Control", flag: Control},
	{name: "Shift", flag: Shift},
	{name: "Alt", flag: Alt},
	{name: "Win", flag: Win},
}

var namedKeys = []struct {
	name string
	code uint32
}{
	{"Space", 0x20}, {"PageUp", 0x21}, {"PageDown", 0x22}, {"End", 0x23}, {"Home", 0x24},
	{"Left", 0x25}, {"Up", 0x26}, {"Right", 0x27}, {"Down", 0x28}, {"Insert", 0x2D}, {"Delete", 0x2E},
}

func Bindings(h profiles.Hotkeys) ([]Binding, error) {
	bindings := make([]Binding, 0, len(actions))
	for id, a := range actions {
		text := a.text(h)
		b, ok := parsed(text)
		if !ok {
			return nil, fmt.Errorf("%w: %s %q", ErrBadHotkey, a.name, text)
		}
		b.Action, b.Field, b.Repeat = id, a.field.name, a.arrow != ""

		if clash := slices.IndexFunc(bindings, b.same); clash >= 0 {
			return nil, fmt.Errorf("%w: %s and %s on %s", ErrSameHotkey, actions[clash].name, a.name, b.Text)
		}
		bindings = append(bindings, b)
	}

	return bindings, nil
}

func Apply(s profiles.Settings, action int, exe string) (profiles.Settings, error) {
	if action < 0 || action >= len(actions) {
		return s, fmt.Errorf("%w: %d", ErrNoAction, action)
	}

	return actions[action].apply(s, exe)
}

func (a action) text(h profiles.Hotkeys) string {
	text := *a.field.access(&h)
	if a.arrow == "" {
		return text
	}

	return text + "+" + a.arrow
}

func (b Binding) Flags() uint32 {
	if b.Repeat {
		return b.Modifiers
	}

	return b.Modifiers | NoRepeat
}

func (b Binding) same(other Binding) bool {
	return b.Modifiers == other.Modifiers && b.Key == other.Key
}

func parsed(text string) (Binding, bool) {
	var b Binding
	key := ""
	for _, token := range strings.Split(text, "+") {
		token = strings.TrimSpace(token)
		if flag, ok := modifierOf(token); ok {
			if b.Modifiers&flag != 0 {
				return Binding{}, false
			}
			b.Modifiers |= flag
			continue
		}

		name, code, ok := keyOf(token)
		if !ok || key != "" {
			return Binding{}, false
		}
		key, b.Key = name, code
	}

	if key == "" || b.Modifiers&(Control|Alt|Win) == 0 {
		return Binding{}, false
	}
	b.Text = textOf(b.Modifiers, key)

	return b, true
}

func modifierOf(token string) (uint32, bool) {
	for _, m := range modifiers {
		if strings.EqualFold(token, m.name) || m.alias != "" && strings.EqualFold(token, m.alias) {
			return m.flag, true
		}
	}

	return 0, false
}

func keyOf(token string) (string, uint32, bool) {
	if len(token) == 1 {
		c := strings.ToUpper(token)[0]
		if 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
			return string(c), uint32(c), true
		}
	}
	if n, ok := numbered(token, "F", 1, 24); ok {
		return "F" + strconv.Itoa(n), 0x70 + uint32(n) - 1, true
	}
	if n, ok := numbered(token, "Numpad", 0, 9); ok {
		return "Numpad" + strconv.Itoa(n), 0x60 + uint32(n), true
	}
	for _, k := range namedKeys {
		if strings.EqualFold(token, k.name) {
			return k.name, k.code, true
		}
	}

	return "", 0, false
}

func numbered(token, prefix string, lowest, highest int) (int, bool) {
	if len(token) < len(prefix) || !strings.EqualFold(token[:len(prefix)], prefix) {
		return 0, false
	}

	n, err := strconv.Atoi(token[len(prefix):])
	if err != nil || n < lowest || n > highest {
		return 0, false
	}

	return n, true
}

func textOf(flags uint32, key string) string {
	parts := []string{}
	for _, m := range modifiers {
		if flags&m.flag != 0 {
			parts = append(parts, m.name)
		}
	}
	if key != "" {
		parts = append(parts, key)
	}

	return strings.Join(parts, "+")
}

func toggled(s profiles.Settings, _ string) (profiles.Settings, error) {
	s.Visible = !s.Visible

	return s, nil
}

func nudged(dx, dy int) func(profiles.Settings, string) (profiles.Settings, error) {
	return func(s profiles.Settings, exe string) (profiles.Settings, error) {
		return s.Edited(exe, func(g profiles.Game) profiles.Game {
			g.Offset = profiles.Offset{X: g.Offset.X + dx, Y: g.Offset.Y + dy}
			return g
		})
	}
}

func centred(s profiles.Settings, exe string) (profiles.Settings, error) {
	return s.Edited(exe, func(g profiles.Game) profiles.Game {
		g.Offset = profiles.Offset{}
		return g
	})
}

func cycled(s profiles.Settings, exe string) (profiles.Settings, error) {
	return s.Edited(exe, func(g profiles.Game) profiles.Game {
		g.Crosshair, _ = s.NextSaved(g.Crosshair)
		return g
	})
}
