package hotkeys

import (
	"slices"
	"strings"

	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

// Offered in place of a taken hotkey, most familiar first. Ctrl+Alt with an arrow turns the
// screen on many computers with Intel graphics, so the nudge fields never get it.
var offered = []uint32{
	Control | Shift,
	Control | Shift | Alt,
	Control | Alt,
	Shift | Alt,
	Control | Win,
	Control | Shift | Win,
}

// Suggest keeps the key of a taken hotkey and looks for modifiers that are free with it and
// clash with no other action.
func Suggest(h profiles.Hotkeys, name string, taken func([]Binding) []Binding) (string, bool) {
	i := slices.IndexFunc(fields, func(f field) bool { return f.name == name })
	if i < 0 {
		return "", false
	}
	f := fields[i]

	for _, candidate := range f.candidates(*f.access(&h)) {
		*f.access(&h) = candidate
		bindings, err := Bindings(h)
		if err != nil {
			continue
		}
		if len(taken(ownedBy(bindings, name))) == 0 {
			return candidate, true
		}
	}

	return "", false
}

func (f field) candidates(current string) []string {
	b, _ := parsed(current)
	key := b.Text[strings.LastIndex(b.Text, "+")+1:]

	var texts []string
	for _, flags := range offered {
		if f.arrows && flags == Control|Alt {
			continue
		}
		texts = append(texts, textOf(flags, key))
	}

	return texts
}

func ownedBy(bindings []Binding, name string) []Binding {
	var owned []Binding
	for _, b := range bindings {
		if b.Field == name {
			owned = append(owned, b)
		}
	}

	return owned
}
