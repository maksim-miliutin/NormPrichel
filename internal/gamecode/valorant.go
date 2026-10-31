package gamecode

import (
	"encoding/hex"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

// Valorant's colours in the order of their numbers; 8 takes the colour written after u.
var valorantColours = []crosshair.RGB{
	{R: 0xFF, G: 0xFF, B: 0xFF},
	{R: 0x00, G: 0xFF, B: 0x00},
	{R: 0x7F, G: 0xFF, B: 0x00},
	{R: 0xDF, G: 0xFF, B: 0x00},
	{R: 0xFF, G: 0xFF, B: 0x00},
	{R: 0x00, G: 0xFF, B: 0xFF},
	{R: 0xFF, G: 0x00, B: 0xFF},
	{R: 0xFF, G: 0x00, B: 0x00},
}

const customColour = 8

// A Valorant code is a version and then keys and values between semicolons; P, A and S open the
// primary, aiming and sniper crosshairs, and keys before them belong to none. Only P is drawn.
func valorant(text string) (crosshair.Style, error) {
	atoms := strings.Split(strings.TrimSuffix(text, ";"), ";")
	primary, section := settings{}, ""
	for i := 1; i < len(atoms); {
		if slices.Contains([]string{"P", "A", "S"}, atoms[i]) {
			section = atoms[i]
			i++
			continue
		}
		if i+1 >= len(atoms) || atoms[i] == "" {
			return crosshair.Style{}, fmt.Errorf("%w: %q", ErrBroken, text)
		}
		if section == "P" {
			primary[atoms[i]] = atoms[i+1]
		}
		i += 2
	}

	s, err := primary.style()
	if err != nil {
		return crosshair.Style{}, fmt.Errorf("%w: %q: %w", ErrBroken, text, err)
	}

	return s, nil
}

type settings map[string]string

// A key the code leaves out has the value the game starts with; these are Valorant's own.
func (p settings) style() (crosshair.Style, error) {
	r := reader{values: p}
	colour, alpha := r.colour()
	share := func(key string, fallback, over float64) int {
		return int(math.Round(100 * over * min(max(r.number(key, fallback), 0), 1)))
	}
	pixels := func(key string, fallback float64) int {
		return int(math.Round(r.number(key, fallback)))
	}
	lines := func(prefix string, opacity, length, thickness, offset float64) part {
		values := map[string]int{
			"opacity": share(prefix+"a", opacity, alpha), "length": pixels(prefix+"l", length),
			"thickness": pixels(prefix+"t", thickness), "gap": pixels(prefix+"o", offset),
		}
		shown := r.number(prefix+"b", 1) != 0 && values["length"] > 0 && values["thickness"] > 0
		return part{name: map[string]crosshair.Part{"0": crosshair.InnerPart, "1": crosshair.OuterPart}[prefix], shown: shown, values: values}
	}

	outline := 0
	if r.number("h", 1) != 0 {
		outline = pixels("t", 1)
	}
	parts := []part{
		lines("0", 0.8, 6, 2, 3),
		lines("1", 0.35, 2, 2, 10),
		{name: crosshair.DotPart, shown: r.number("d", 0) != 0, values: map[string]int{"opacity": share("a", 1, alpha), "dot": pixels("z", 2)}},
		{name: crosshair.RingPart},
		{name: crosshair.OutlinePart, values: map[string]int{"outline": outline, "outlineOpacity": share("o", 0.5, 1)}},
	}
	if r.err != nil {
		return crosshair.Style{}, r.err
	}

	return built(colour, crosshair.RGB{}, parts)
}

type reader struct {
	values settings
	err    error
}

func (r *reader) number(key string, fallback float64) float64 {
	text, written := r.values[key]
	if !written {
		return fallback
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		r.err = fmt.Errorf("%s is %q", key, text)
		return fallback
	}

	return value
}

// The custom colour is written RRGGBBAA, its last byte an opacity over every part.
func (r *reader) colour() (crosshair.RGB, float64) {
	index := int(r.number("c", 0))
	if index >= 0 && index < len(valorantColours) {
		return valorantColours[index], 1
	}
	text := r.values["u"]
	channels, err := hex.DecodeString(text)
	if index != customColour || err != nil || len(channels) < 3 || len(channels) > 4 {
		r.err = fmt.Errorf("the colour is c %d u %q", index, text)
		return valorantColours[0], 1
	}
	alpha := 1.0
	if len(channels) == 4 {
		alpha = float64(channels[3]) / 255
	}

	return crosshair.RGB{R: channels[0], G: channels[1], B: channels[2]}, alpha
}
