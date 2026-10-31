package crosshair

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type field struct {
	write func(s Style) string
	read  func(s *Style, text string) bool
}

// Every field of a built code after its version, written and read in this one order.
var builtFields = slices.Concat(
	[]field{
		colourField(func(s *Style) *RGB { return &s.Color }),
		numberField(func(s *Style) *int { return &s.Outline.Width }),
		colourField(func(s *Style) *RGB { return &s.Outline.Color }),
		numberField(func(s *Style) *int { return &s.Parts.OutlineOpacity }),
		{write: cornersText, read: readCorners},
		{write: outlineText, read: readOutline},
	},
	linesFields(func(s *Style) *Lines { return &s.Parts.Inner }),
	linesFields(func(s *Style) *Lines { return &s.Parts.Outer }),
	[]field{
		flagField(func(s *Style) *bool { return &s.Parts.Dot.On }),
		numberField(func(s *Style) *int { return &s.Parts.Dot.Opacity }),
		numberField(func(s *Style) *int { return &s.Parts.Dot.Size }),
		flagField(func(s *Style) *bool { return &s.Parts.Ring.On }),
		numberField(func(s *Style) *int { return &s.Parts.Ring.Opacity }),
		numberField(func(s *Style) *int { return &s.Parts.Ring.Radius }),
		numberField(func(s *Style) *int { return &s.Parts.Ring.Thickness }),
	},
)

func linesFields(at func(s *Style) *Lines) []field {
	return []field{
		flagField(func(s *Style) *bool { return &at(s).On }),
		numberField(func(s *Style) *int { return &at(s).Opacity }),
		numberField(func(s *Style) *int { return &at(s).Length }),
		numberField(func(s *Style) *int { return &at(s).Thickness }),
		numberField(func(s *Style) *int { return &at(s).Gap }),
		{
			write: func(s Style) string { return armsText(at(&s).Hidden) },
			read: func(s *Style, text string) bool {
				hidden, ok := armsOf(text)
				at(s).Hidden = hidden
				return ok
			},
		},
	}
}

func builtCode(c Style) string {
	texts := []string{builtVersion}
	for _, f := range builtFields {
		texts = append(texts, f.write(c))
	}

	return strings.Join(texts, ".")
}

// The ready shape is not in the code, so a built crosshair read from one keeps the default shape.
func parseBuilt(fields []string, text string) (Style, error) {
	s := Default()
	s.Built = true
	for i, f := range builtFields {
		if !f.read(&s, fields[i]) {
			return Style{}, fmt.Errorf("%w: %q", ErrBadCode, text)
		}
	}

	return s.clamped(), nil
}

func numberField(at func(s *Style) *int) field {
	return field{
		write: func(s Style) string { return strconv.Itoa(*at(&s)) },
		read: func(s *Style, text string) bool {
			values, ok := numbers([]string{text})
			if ok {
				*at(s) = values[0]
			}
			return ok
		},
	}
}

func flagField(at func(s *Style) *bool) field {
	return field{
		write: func(s Style) string { return map[bool]string{false: "0", true: "1"}[*at(&s)] },
		read: func(s *Style, text string) bool {
			*at(s) = text == "1"
			return text == "0" || text == "1"
		},
	}
}

func colourField(at func(s *Style) *RGB) field {
	return field{
		write: func(s Style) string { return hexOf(*at(&s)) },
		read: func(s *Style, text string) bool {
			colour, err := colourOf(text)
			*at(s) = colour
			return err == nil
		},
	}
}

func readCorners(s *Style, text string) bool {
	corners, rounding, ok := cornersOf(text)
	s.Corners, s.Rounding = corners, rounding

	return ok
}

func readOutline(s *Style, text string) bool {
	outline := strings.ToLower(text)
	s.Outline.Shadow = outline == "shadow"

	return outline == "line" || outline == "shadow"
}

// The arms go up, right, down and left, a 1 for each shown and a 0 for each hidden.
func armsText(hidden Arms) string {
	var text strings.Builder
	for _, h := range []bool{hidden.Up, hidden.Right, hidden.Down, hidden.Left} {
		text.WriteString(map[bool]string{false: "1", true: "0"}[h])
	}

	return text.String()
}

func armsOf(text string) (Arms, bool) {
	if len(text) != 4 || strings.Trim(text, "01") != "" {
		return Arms{}, false
	}

	return Arms{Up: text[0] == '0', Right: text[1] == '0', Down: text[2] == '0', Left: text[3] == '0'}, true
}
