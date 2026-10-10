package profiles_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

const picture = "9f2c0000000000000000000000000000000000000000000000000000000000e1.png"

var specDefaults = profiles.Settings{
	Version: 1,
	Visible: true,
	Hotkeys: profiles.Hotkeys{
		Toggle:   "Ctrl+Shift+X",
		Nudge:    "Ctrl+Shift",
		NudgeFar: "Ctrl+Shift+Alt",
		Center:   "Ctrl+Shift+Home",
		Next:     "Ctrl+Shift+C",
	},
	Games: []profiles.Game{},
	Saved: []profiles.Saved{},
}

var specStyle = crosshair.Style{
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

func TestTheSpecExampleReads(t *testing.T) {
	example := `{
		"version": 1,
		"visible": true,
		"hotkeys": {
			"toggle": "Ctrl+Shift+X",
			"nudge": "Ctrl+Shift",
			"nudgeFar": "Ctrl+Shift+Alt",
			"center": "Ctrl+Shift+Home",
			"next": "Ctrl+Shift+C"
		},
		"games": [
			{
				"exe": "stalzone.exe",
				"name": "STALZONE",
				"enabled": true,
				"crosshair": {
					"shape": "cross-dot",
					"color": "#00FF66",
					"opacity": 100,
					"length": 8,
					"thickness": 2,
					"gap": 3,
					"dot": 2,
					"outline": { "width": 1, "color": "#000000" }
				},
				"offset": { "x": 0, "y": 0 }
			}
		],
		"saved": [
			{ "name": "Своя картинка", "image": "` + picture + `", "scale": 100 }
		]
	}`

	got, err := profiles.Parse([]byte(example))
	if err != nil {
		t.Fatal(err)
	}

	want := specDefaults
	want.Games = []profiles.Game{{
		Exe:       "stalzone.exe",
		Name:      "STALZONE",
		Enabled:   true,
		Crosshair: profiles.Crosshair{Style: specStyle, Scale: 100},
	}}
	want.Saved = []profiles.Saved{{
		Name:      "Своя картинка",
		Crosshair: profiles.Crosshair{Style: specStyle, Image: picture, Scale: 100},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%+v\nwant\n%+v", got, want)
	}
}

func TestAnEmptyObjectReadsAsTheSpecDefaults(t *testing.T) {
	got, err := profiles.Parse([]byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, specDefaults) {
		t.Errorf("got %+v, want %+v", got, specDefaults)
	}
}

func TestAByteOrderMarkFromNotepadOrPowerShellIsSkipped(t *testing.T) {
	got, err := profiles.Parse([]byte("\xEF\xBB\xBF{ \"games\": [ { \"exe\": \"stalzone.exe\" } ] }"))
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := got.Game("stalzone.exe"); !ok {
		t.Errorf("got %+v, want the game listed after the mark", got)
	}
}

func TestOmittedFieldsTakeTheirDefaults(t *testing.T) {
	got, err := profiles.Parse([]byte(`{
		"hotkeys": { "toggle": "Ctrl+Alt+F9" },
		"games": [ { "exe": "game.exe", "crosshair": { "shape": "circle", "opacity": 40 } } ]
	}`))
	if err != nil {
		t.Fatal(err)
	}

	if got.Hotkeys.Toggle != "Ctrl+Alt+F9" || got.Hotkeys.Next != "Ctrl+Shift+C" {
		t.Errorf("hotkeys %+v, want the one given and the rest by default", got.Hotkeys)
	}

	game := got.Games[0]
	if !game.Enabled {
		t.Error("a game without an enabled flag is off, want it on")
	}

	want := specStyle
	want.Shape, want.Opacity = crosshair.Circle, 40
	if game.Crosshair.Style != want {
		t.Errorf("crosshair %+v, want %+v", game.Crosshair.Style, want)
	}
}

func TestOffsetsStayWithin500PixelsOfTheCentre(t *testing.T) {
	got, err := profiles.Parse([]byte(`{ "games": [ { "exe": "game.exe", "offset": { "x": 900, "y": -501 } } ] }`))
	if err != nil {
		t.Fatal(err)
	}

	if want := (profiles.Offset{X: 500, Y: -500}); got.Games[0].Offset != want {
		t.Errorf("offset %+v, want %+v", got.Games[0].Offset, want)
	}
}

func TestAGameListedTwiceKeepsItsFirstEntry(t *testing.T) {
	got, err := profiles.Parse([]byte(`{ "games": [
		{ "exe": "Stalzone.exe", "name": "first" },
		{ "exe": "" },
		{ "exe": "STALZONE.EXE", "name": "second" }
	] }`))
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Games) != 1 || got.Games[0].Name != "first" {
		t.Errorf("games %+v, want only the first entry", got.Games)
	}
}

func TestAGameIsFoundWhateverTheCaseOfItsExe(t *testing.T) {
	settings, err := profiles.Parse([]byte(`{ "games": [ { "exe": "stalzone.exe", "name": "STALZONE" } ] }`))
	if err != nil {
		t.Fatal(err)
	}

	if game, ok := settings.Game("StalZone.EXE"); !ok || game.Name != "STALZONE" {
		t.Errorf("got %+v, %v; want the STALZONE entry", game, ok)
	}
	if _, ok := settings.Game("notepad.exe"); ok {
		t.Error("found a game that is not on the list")
	}
}

func TestBrokenFilesAreRefused(t *testing.T) {
	cases := map[string]string{
		"not json":           `{ "version": 1, `,
		"a number as text":   `{ "games": [ { "exe": "a.exe", "crosshair": { "opacity": "full" } } ] }`,
		"a colour by name":   `{ "games": [ { "exe": "a.exe", "crosshair": { "color": "green" } } ] }`,
		"an unknown shape":   `{ "games": [ { "exe": "a.exe", "crosshair": { "shape": "star" } } ] }`,
		"a newer version":    `{ "version": 2 }`,
		"a picture path":     `{ "saved": [ { "name": "x", "image": "../../secret.png" } ] }`,
		"an upper case hash": `{ "games": [ { "exe": "a.exe", "crosshair": { "image": "` + strings.ToUpper(picture[:64]) + `.png" } } ] }`,
	}

	for name, text := range cases {
		if _, err := profiles.Parse([]byte(text)); !errors.Is(err, profiles.ErrBroken) {
			t.Errorf("%s: got %v, want ErrBroken", name, err)
		}
	}
}

func TestSettingsSurviveARoundTrip(t *testing.T) {
	circle := specStyle
	circle.Shape, circle.Color, circle.Gap = crosshair.Circle, crosshair.RGB{R: 200, G: 10, B: 10}, 0

	settings := specDefaults
	settings.Visible = false
	settings.Hotkeys.Center = "Ctrl+Shift+End"
	settings.Games = []profiles.Game{
		{Exe: "stalzone.exe", Name: "STALZONE", Enabled: true, Crosshair: profiles.Crosshair{Style: circle, Scale: 100}, Offset: profiles.Offset{X: -3, Y: 12}},
		{Exe: "other.exe", Name: "Другая", Enabled: false, Crosshair: profiles.Crosshair{Style: specStyle, Image: picture, Scale: 250}},
	}
	settings.Saved = []profiles.Saved{{Name: "Красный круг", Crosshair: profiles.Crosshair{Style: circle, Scale: 100}}}

	text, err := settings.Encode()
	if err != nil {
		t.Fatal(err)
	}

	back, err := profiles.Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, settings) {
		t.Errorf("came back as\n%+v\nfrom\n%s", back, text)
	}
}
