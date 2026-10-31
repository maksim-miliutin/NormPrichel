package gamecode

import (
	"errors"
	"strings"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

var (
	ErrUnknown = errors.New("gamecode: this is not a crosshair code of CS2 or Valorant")
	ErrBroken  = errors.New("gamecode: the crosshair code is damaged")
)

func Read(text string, screenHeight int) (crosshair.Style, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "0;") || text == "0" {
		return valorant(text)
	}
	if !strings.HasPrefix(text, "CS") {
		return crosshair.Style{}, ErrUnknown
	}
	c, err := decodeCS2(text)
	if err != nil {
		return crosshair.Style{}, err
	}

	return c.drawn(screenHeight)
}

// A part is set through the crosshair's own knobs, which hold every value to its range.
type part struct {
	name   crosshair.Part
	shown  bool
	values map[string]int
	hidden []crosshair.Arm
}

func built(colour, edge crosshair.RGB, parts []part) (crosshair.Style, error) {
	s := crosshair.Default()
	s.Color, s.Outline = colour, crosshair.Outline{Color: edge}
	s = s.Build()
	for _, p := range parts {
		s = s.WithPartShown(p.name, p.shown)
		for name, value := range p.values {
			changed, err := s.WithPartValue(p.name, name, value)
			if err != nil {
				return crosshair.Style{}, err
			}
			s = changed
		}
		for _, arm := range p.hidden {
			s = s.WithArm(p.name, arm, false)
		}
	}

	return s, nil
}
