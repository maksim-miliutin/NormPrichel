package hotkeys

import (
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var ErrNoField = errors.New("hotkeys: there is no such hotkey")

var arrowKeys = []string{"Left", "Right", "Up", "Down"}

func Fields() []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.name
	}

	return names
}

func Get(h profiles.Hotkeys, name string) string {
	f, ok := fieldNamed(name)
	if !ok {
		return ""
	}

	return *f.access(&h)
}

func Set(h profiles.Hotkeys, name, text string) (profiles.Hotkeys, error) {
	f, ok := fieldNamed(name)
	if !ok {
		return h, fmt.Errorf("%w: %q", ErrNoField, name)
	}
	*f.access(&h) = text

	return h, nil
}

func Arrows(name string) bool {
	f, ok := fieldNamed(name)

	return ok && f.arrows
}

// A press counts only with Ctrl, Alt or Win held: without them the field takes typing, so a
// hotkey the window itself would swallow can still be written in by hand.
func Pressed(name string, modifiers, key uint32) (string, bool) {
	keyName, ok := nameOf(key)
	if !ok || modifiers&(Control|Alt|Win) == 0 {
		return "", false
	}
	if !Arrows(name) {
		return textOf(modifiers, keyName), true
	}
	if !slices.Contains(arrowKeys, keyName) {
		return "", false
	}

	return textOf(modifiers, ""), true
}

func nameOf(code uint32) (string, bool) {
	switch {
	case 'A' <= code && code <= 'Z', '0' <= code && code <= '9':
		return string(rune(code)), true
	case 0x70 <= code && code <= 0x87:
		return "F" + strconv.Itoa(int(code-0x70+1)), true
	case 0x60 <= code && code <= 0x69:
		return "Numpad" + strconv.Itoa(int(code-0x60)), true
	}
	for _, k := range namedKeys {
		if k.code == code {
			return k.name, true
		}
	}

	return "", false
}

func fieldNamed(name string) (field, bool) {
	i := slices.IndexFunc(fields, func(f field) bool { return f.name == name })
	if i < 0 {
		return field{}, false
	}

	return fields[i], true
}
