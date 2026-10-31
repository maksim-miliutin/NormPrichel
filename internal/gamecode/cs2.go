package gamecode

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

var ErrCSGO = errors.New("gamecode: a CS:GO crosshair code from before CS2 counted in pixels, which CS2 no longer takes either")

// Valve leaves out I, g, l, 0 and 1, the characters easy to mix up when a code is typed again.
const dictionary = "ABCDEFGHJKLMNOPQRSTUVWXYZabcdefhijkmnopqrstuvwxyz23456789"

var (
	cs2Code    = regexp.MustCompile(`^CS([` + dictionary + `]{44})$`)
	legacyCode = regexp.MustCompile(`^CSGO((?:-?[` + dictionary + `]{5}){5})$`)
)

type cs2Crosshair struct {
	style                  int
	dot, tee               bool
	outline                int // 0 none, 1 all round, 2 on the top and left edges only
	colour, edge           crosshair.RGB
	alpha, edgeAlpha       int // out of 255
	gap, length, thickness int // pixels on a screen height pixels high
	height                 int
}

func decodeCS2(text string) (cs2Crosshair, error) {
	if found := legacyCode.FindStringSubmatch(text); found != nil {
		return legacy(text, strings.ReplaceAll(found[1], "-", ""))
	}
	found := cs2Code.FindStringSubmatch(text)
	if found == nil {
		return cs2Crosshair{}, fmt.Errorf("%w: %q", ErrUnknown, text)
	}
	b, ok := checked(found[1], 32)
	if !ok || b[1] != 1 {
		return cs2Crosshair{}, fmt.Errorf("%w: %q", ErrBroken, text)
	}

	c := cs2Crosshair{
		style: min(int(b[4]&0x1f), 9), dot: b[4]&0x40 != 0, tee: b[4]&0x80 != 0,
		colour: crosshair.RGB{R: b[5], G: b[6], B: b[7]}, alpha: int(b[8]),
		edge: crosshair.RGB{R: b[9], G: b[10], B: b[11]}, edgeAlpha: int(b[12]),
		thickness: min(int(b[13]&0x3f), 32), outline: min(int(b[13]>>6), 2),
		gap:    min(max(int(int16(uint16(b[14])|uint16(b[15])<<8)), -3840), 3840),
		length: int(b[16]), height: int(b[2]) | int(b[3])<<8,
	}

	return c.heightened(text)
}

// The bytes 10 to 13 hold their fields bit by bit, the lowest bit first.
func legacy(text, chars string) (cs2Crosshair, error) {
	b, ok := checked(chars, 18)
	if !ok {
		return cs2Crosshair{}, fmt.Errorf("%w: %q", ErrBroken, text)
	}
	if b[1] == 1 {
		return cs2Crosshair{}, fmt.Errorf("%w: %q", ErrCSGO, text)
	}
	if b[1] != 3 && b[1] != 4 {
		return cs2Crosshair{}, fmt.Errorf("%w: %q", ErrBroken, text)
	}

	bits := uint32(b[10]) | uint32(b[11])<<8 | uint32(b[12])<<16 | uint32(b[13])<<24
	c := cs2Crosshair{
		style: int(b[2] & 0xf), dot: b[2]&0x40 != 0, tee: b[2]&0x80 != 0,
		colour: crosshair.RGB{R: b[3], G: b[4], B: b[5]}, alpha: int(b[6]), edgeAlpha: 255,
		gap: int(b[7]), length: int(b[8]), thickness: int(bits >> 23 & 0x1f),
		outline: int(bits >> 28 & 3), height: int(b[14]) | int(b[15])<<8,
	}
	if b[1] == 3 {
		c.outline = int(b[2] >> 5 & 1)
	}

	return c.heightened(text)
}

// CS2 refuses a code made for no screen and draws one made for less than 240 pixels at 240.
func (c cs2Crosshair) heightened(text string) (cs2Crosshair, error) {
	if c.height == 0 {
		return cs2Crosshair{}, fmt.Errorf("%w: %q", ErrBroken, text)
	}
	c.height = max(c.height, 240)

	return c, nil
}

// A code holds one big number in base 57, its last character the most significant, read as
// bytes from the most significant; the first byte is the sum of the others.
func checked(chars string, size int) ([]byte, bool) {
	number, base := new(big.Int), big.NewInt(int64(len(dictionary)))
	for i := len(chars) - 1; i >= 0; i-- {
		number.Mul(number, base)
		number.Add(number, big.NewInt(int64(strings.IndexByte(dictionary, chars[i]))))
	}
	if number.BitLen() > 8*size {
		return nil, false
	}
	b := number.FillBytes(make([]byte, size))
	sum := 0
	for _, v := range b[1:] {
		sum += int(v)
	}

	return b, byte(sum) == b[0]
}
