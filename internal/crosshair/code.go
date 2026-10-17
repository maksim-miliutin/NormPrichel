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
const codeVersion = "np1"

func (s Style) Code() string {
	c := s.clamped()

	return fmt.Sprintf("%s.%s.%s.%d.%d.%d.%d.%d.%d.%d.%s", codeVersion, c.Shape, hexOf(c.Color),
		c.Opacity, c.Length, c.Thickness, c.Gap, c.Radius, c.DotSize, c.Outline.Width, hexOf(c.Outline.Color))
}

func ParseCode(text string) (Style, error) {
	fields := strings.Split(strings.TrimSpace(text), ".")
	if len(fields) != 11 || !strings.EqualFold(fields[0], codeVersion) || !slices.Contains(Shapes(), Shape(fields[1])) {
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
		Gap: values[3], Radius: values[4], DotSize: values[5], Outline: Outline{Width: values[6], Color: edge},
	}

	return s.clamped(), nil
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
