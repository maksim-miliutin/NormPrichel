package gamecode

import (
	"errors"
	"math/big"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

// Codes and values from the tests of github.com/akiver/csgo-sharecode, whose author checked the
// codes CS2 made against the console values the game wrote for them.
var cs2Samples = []struct {
	code string
	want cs2Crosshair
}{
	{"CSGO-MnUCC-89iG7-2cVar-wy7Yn-amCpF", cs2Crosshair{style: 6, outline: 1, dot: true, colour: rgb(252, 15, 192), alpha: 255, edgeAlpha: 255, gap: 4, length: 8, thickness: 3, height: 1080}},
	{"CSGO-tsJdC-2PFX6-fw8qD-6GDaX-rYCUM", cs2Crosshair{style: 4, dot: true, colour: rgb(0, 40, 255), alpha: 255, edgeAlpha: 255, gap: 0, length: 0, thickness: 3, height: 768}},
	{"CSGO-vcmKL-TWb8f-ddSDD-qM2Xx-T66VH", cs2Crosshair{style: 4, outline: 1, dot: true, colour: rgb(172, 14, 61), alpha: 255, edgeAlpha: 255, gap: 0, length: 4, thickness: 5, height: 1080}},
	{"CSGO-cA4U9-hiJwT-Wo9NA-YmTS8-WfH4C", cs2Crosshair{style: 6, outline: 1, colour: rgb(0, 255, 14), alpha: 255, edgeAlpha: 255, gap: 2, length: 2, thickness: 2, height: 960}},
	{"CSGO-p4NqQ-mV2es-p2UF3-Q9HWe-fVmoE", cs2Crosshair{style: 3, outline: 1, colour: rgb(0, 255, 0), alpha: 255, edgeAlpha: 255, gap: 4, length: 8, thickness: 1, height: 1080}},
	{"CSGO-hLbCn-69VT6-Bok83-9MOqW-SWzwQ", cs2Crosshair{style: 4, colour: rgb(50, 250, 50), alpha: 255, edgeAlpha: 255, gap: 4, length: 8, thickness: 2, height: 1080}},
	{"CSGO-G8oAC-RyvWc-Hi3CZ-voSwn-QJbfE", cs2Crosshair{style: 8, outline: 1, dot: true, colour: rgb(124, 57, 57), alpha: 255, edgeAlpha: 255, gap: 25, length: 7, thickness: 20, height: 768}},
	{"CSGO-sP6xU-TSyN9-sZcO5-2D48M-UppkP", cs2Crosshair{style: 2, dot: true, colour: rgb(255, 0, 0), alpha: 255, edgeAlpha: 255, gap: 0, length: 5, thickness: 1, height: 768}},
	{"CSvbPubOq37zTGqtsPTP5QTrp5CB4xFXiKRLfzJsm49ZRe", cs2Crosshair{style: 2, dot: true, colour: rgb(255, 0, 0), alpha: 255, edgeAlpha: 255, gap: 0, length: 5, thickness: 1, height: 768}},
	{"CSAB6Oby2JUDOrBRy4jRcsR6JOQdhbszKF9GuKQGhKVwAc", cs2Crosshair{style: 9, tee: true, outline: 2, colour: rgb(10, 20, 30), alpha: 200, edge: rgb(40, 50, 60), edgeAlpha: 128, gap: -7, length: 255, thickness: 12, height: 1440}},
	{"CS7uw3wMkJBJ4rMDLEaYbyfTJFFPvkLAhOmBFi8zOM6H2d", cs2Crosshair{style: 9, outline: 2, colour: rgb(1, 2, 3), alpha: 4, edge: rgb(5, 6, 7), edgeAlpha: 8, gap: -3840, length: 9, thickness: 32, height: 240}},
}

func rgb(r, g, b uint8) crosshair.RGB {
	return crosshair.RGB{R: r, G: g, B: b}
}

func TestEveryCS2CodeDecodesToTheValuesTheGameWrote(t *testing.T) {
	for _, sample := range cs2Samples {
		got, err := decodeCS2(sample.code)
		if err != nil || got != sample.want {
			t.Errorf("%s gives %+v, %v; want %+v", sample.code, got, err, sample.want)
		}
	}
}

func TestADashlessLegacyCodeReadsAsTheDashedOne(t *testing.T) {
	got, err := decodeCS2("CSGOMnUCC89iG72cVarwy7YnamCpF")
	if err != nil || got != cs2Samples[0].want {
		t.Errorf("got %+v, %v; want %+v", got, err, cs2Samples[0].want)
	}
}

func TestAGapPastWhatCS2TakesIsHeldToIt(t *testing.T) {
	got, err := decodeCS2("CSDfPcMpj4e9NRxx6SGiVoFCHzYRYGmb34WHpbYHy6E6id")
	if err != nil || got.gap != 3840 {
		t.Errorf("got gap %d, %v; want 3840", got.gap, err)
	}
}

func TestCodesCS2RefusesAreRefusedForTheirReason(t *testing.T) {
	for _, c := range []struct {
		code string
		want error
	}{
		{"CSGO-Cn37R-YE7vo-pLCAL-aURmZ-z6zkG", ErrCSGO},
		{"CSGO-L9spZ-ihuov-cyhtE-kxbqa-FkBAA", ErrBroken},
		{"CSGO-eRqP8-AkrwM-3Wsqh-pKDTh-eAPtQ", ErrBroken},
		{"CSuZhVyU8icUwjTjopBSVi6ctEbq5GiOXJBRdC6LNLKaFL", ErrBroken},
		{"CSkSCHQKunGr2yPsJr2U3RTd8ycMA2pEs9jWnxK5KjMmWL", ErrBroken},
		{"CSoQhciXBHZUKAbfexZBeMWFKxkrXL5MLumDLj3JYW4snP", ErrBroken},
		{"CSGO-99999-99999-99999-99999-99999", ErrBroken},
		{"CSGO-12345-12345-12345-12345-12345", ErrUnknown},
		{"CSGO-MnUCC-89iG7", ErrUnknown},
		{"CSvbPubOq37zTGqtsPTP5QTrp5CB4xFXiKRLfzJsm49ZR", ErrUnknown},
	} {
		if _, err := decodeCS2(c.code); !errors.Is(err, c.want) {
			t.Errorf("%s gives %v, want %v", c.code, err, c.want)
		}
	}
}

// written spells bytes back as a code, as the game does, so a test can change one byte of a good one.
func written(b []byte, chars int) string {
	number, base := new(big.Int).SetBytes(b), big.NewInt(int64(len(dictionary)))
	out := make([]byte, chars)
	for i := range out {
		rest := new(big.Int)
		number.DivMod(number, base, rest)
		out[i] = dictionary[rest.Int64()]
	}

	return string(out)
}

func TestACodeWhoseSumIsOffIsRefused(t *testing.T) {
	for _, c := range []struct {
		prefix, chars string
		size          int
	}{
		{"CS", "vbPubOq37zTGqtsPTP5QTrp5CB4xFXiKRLfzJsm49ZRe", 32},
		{"CSGO", "MnUCC89iG72cVarwy7YnamCpF", 18},
	} {
		b, ok := checked(c.chars, c.size)
		if !ok || c.prefix+written(b, len(c.chars)) != c.prefix+c.chars {
			t.Fatalf("%s%s does not read back as itself", c.prefix, c.chars)
		}
		b[0]++
		if _, err := decodeCS2(c.prefix + written(b, len(c.chars))); !errors.Is(err, ErrBroken) {
			t.Errorf("%s with its sum off by one gives %v, want ErrBroken", c.prefix+c.chars, err)
		}
	}
}
