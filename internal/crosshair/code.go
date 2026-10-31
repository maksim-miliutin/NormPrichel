package crosshair

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

var ErrBadCode = errors.New("crosshair: this is not a crosshair code")

// A code carries a style in one line through a chat or a web address: dots between the fields,
// since shape names hold dashes, and colours without the # that would cut an address short.
const (
	codeVersion    = "np1"
	roundedVersion = "np2"
	builtVersion   = "np3"
)

var cornerWords = map[string]Corners{"sharp": Sharp, "soft": Soft, "round": Round}

func (s Style) Code() string {
	c := s.clamped()
	if c.Built {
		return builtCode(c)
	}
	code := fmt.Sprintf("%s.%s.%s.%d.%d.%d.%d.%d.%d.%d.%s", codeVersion, c.Shape, hexOf(c.Color),
		c.Opacity, c.Length, c.Thickness, c.Gap, c.Radius, c.DotSize, c.Outline.Width, hexOf(c.Outline.Color))
	// Older copies of the program and the gallery read the first version only.
	if c.Corners == Sharp && c.Rounding == 0 && !c.Outline.Shadow {
		return code
	}

	return roundedVersion + strings.TrimPrefix(code, codeVersion) + "." + cornersText(c) + "." + outlineText(c)
}

func cornersText(c Style) string {
	switch {
	case c.Rounding > 0:
		return strconv.Itoa(c.Rounding)
	case c.Corners != Sharp:
		return string(c.Corners)
	}

	return "sharp"
}

func outlineText(c Style) string {
	if c.Outline.Shadow {
		return "shadow"
	}

	return "line"
}

func ParseCode(text string) (Style, error) {
	fields := strings.Split(strings.TrimSpace(text), ".")
	if len(fields) == len(builtFields)+1 && strings.EqualFold(fields[0], builtVersion) {
		return parseBuilt(fields[1:], text)
	}
	rounded := len(fields) == 13 && strings.EqualFold(fields[0], roundedVersion)
	plain := len(fields) == 11 && strings.EqualFold(fields[0], codeVersion)
	if !rounded && !plain || !slices.Contains(Shapes(), Shape(fields[1])) {
		return Style{}, fmt.Errorf("%w: %q", ErrBadCode, text)
	}
	corners, rounding, outline, known := Sharp, 0, "line", true
	if rounded {
		corners, rounding, known = cornersOf(fields[11])
		outline = strings.ToLower(fields[12])
	}
	if !known || outline != "line" && outline != "shadow" {
		return Style{}, fmt.Errorf("%w: %q", ErrBadCode, text)
	}

	values, counted := numbers(fields[3:10])
	colour, colourErr := colourOf(fields[2])
	edge, edgeErr := colourOf(fields[10])
	if !counted || colourErr != nil || edgeErr != nil {
		return Style{}, fmt.Errorf("%w: %q", ErrBadCode, text)
	}
	s := Style{
		Shape: Shape(fields[1]), Color: colour, Opacity: values[0], Length: values[1], Thickness: values[2],
		Gap: values[3], Radius: values[4], DotSize: values[5], Corners: corners, Rounding: rounding,
		Outline: Outline{Width: values[6], Color: edge, Shadow: outline == "shadow"},
	}

	return s.clamped(), nil
}

// The corners are a word or the share of the way to round, in percent.
func cornersOf(field string) (Corners, int, bool) {
	if corners, known := cornerWords[strings.ToLower(field)]; known {
		return corners, 0, true
	}
	share, counted := numbers([]string{field})
	if !counted {
		return Sharp, 0, false
	}

	return Sharp, share[0], true
}

func numbers(fields []string) ([]int, bool) {
	values := make([]int, len(fields))
	for i, field := range fields {
		value, err := strconv.Atoi(field)
		if err != nil || len(field) > 3 || strings.Trim(field, "0123456789") != "" {
			return nil, false
		}
		values[i] = value
	}

	return values, true
}

func colourOf(field string) (RGB, error) {
	var c RGB
	err := c.UnmarshalText([]byte("#" + field))

	return c, err
}

func hexOf(c RGB) string {
	return fmt.Sprintf("%02X%02X%02X", c.R, c.G, c.B)
}
