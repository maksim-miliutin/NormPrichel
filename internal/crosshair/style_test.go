package crosshair_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

func TestColoursWriteAsUpperCaseHex(t *testing.T) {
	text, err := crosshair.RGB{R: 0x00, G: 0xFF, B: 0x66}.MarshalText()
	if err != nil {
		t.Fatal(err)
	}

	if string(text) != "#00FF66" {
		t.Errorf("got %q, want #00FF66", text)
	}
}

func TestColoursReadInEitherCase(t *testing.T) {
	for _, text := range []string{"#00ff66", "#00FF66", "#00Ff66"} {
		var got crosshair.RGB
		if err := got.UnmarshalText([]byte(text)); err != nil {
			t.Fatalf("%s: %v", text, err)
		}

		if want := (crosshair.RGB{R: 0x00, G: 0xFF, B: 0x66}); got != want {
			t.Errorf("%s: got %v, want %v", text, got, want)
		}
	}
}

func TestMalformedColoursAreRefused(t *testing.T) {
	for _, text := range []string{"", "00FF66", "x00FF66", "#0F6", "#GG0000", "#00FF667", "#00FF6"} {
		var got crosshair.RGB
		if err := got.UnmarshalText([]byte(text)); !errors.Is(err, crosshair.ErrBadColour) {
			t.Errorf("%q: got %v, want ErrBadColour", text, err)
		}
	}
}

func TestEveryKeyReachesItsField(t *testing.T) {
	text := `{
		"shape": "circle-dot",
		"color": "#010203",
		"opacity": 41,
		"length": 42,
		"thickness": 13,
		"gap": 14,
		"radius": 45,
		"dot": 6,
		"outline": { "width": 3, "color": "#0A0B0C" }
	}`

	var got crosshair.Style
	if err := json.Unmarshal([]byte(text), &got); err != nil {
		t.Fatal(err)
	}

	want := crosshair.Style{
		Shape:     crosshair.CircleDot,
		Color:     crosshair.RGB{R: 1, G: 2, B: 3},
		Opacity:   41,
		Length:    42,
		Thickness: 13,
		Gap:       14,
		Radius:    45,
		DotSize:   6,
		Outline:   crosshair.Outline{Width: 3, Color: crosshair.RGB{R: 10, G: 11, B: 12}},
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestTheSpecExampleKeepsTheDefaultsItOmits(t *testing.T) {
	example := `{
		"shape": "cross-dot",
		"color": "#00FF66",
		"opacity": 100,
		"length": 8,
		"thickness": 2,
		"gap": 3,
		"dot": 2,
		"outline": { "width": 1, "color": "#000000" }
	}`

	got := crosshair.Default()
	if err := json.Unmarshal([]byte(example), &got); err != nil {
		t.Fatal(err)
	}

	want := crosshair.Style{
		Shape:     crosshair.CrossDot,
		Color:     crosshair.RGB{R: 0x00, G: 0xFF, B: 0x66},
		Opacity:   100,
		Length:    8,
		Thickness: 2,
		Gap:       3,
		Radius:    10,
		DotSize:   2,
		Outline:   crosshair.Outline{Width: 1, Color: crosshair.RGB{}},
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestAStyleSurvivesJSON(t *testing.T) {
	style := crosshair.Style{
		Shape:     crosshair.Circle,
		Color:     crosshair.RGB{R: 1, G: 2, B: 3},
		Opacity:   55,
		Length:    7,
		Thickness: 5,
		Gap:       0,
		Radius:    33,
		DotSize:   9,
		Outline:   crosshair.Outline{Width: 3, Color: crosshair.RGB{R: 250, G: 128, B: 4}},
	}

	text, err := json.Marshal(style)
	if err != nil {
		t.Fatal(err)
	}

	var back crosshair.Style
	if err := json.Unmarshal(text, &back); err != nil {
		t.Fatal(err)
	}
	if back != style {
		t.Errorf("came back as %+v from %s", back, text)
	}
}
