package tray_test

import (
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/tray"
)

func TestRussianWindowsGetsRussianAndAnyOtherEnglish(t *testing.T) {
	cases := map[uint16]string{
		0x0419: "ru",
		0x0819: "ru",
		0x0409: "en",
		0x0809: "en",
		0x0422: "en",
		0x0000: "en",
	}

	for id, want := range cases {
		if got := tray.Language(id); got != want {
			t.Errorf("language %#04x: got %s, want %s", id, got, want)
		}
	}
}

func TestTheMenuIsWordedInBothLanguages(t *testing.T) {
	for _, language := range []string{"ru", "en"} {
		l := tray.LabelsFor(language)
		if l.Settings == "" || l.Toggle == "" || l.Autostart == "" || l.Exit == "" {
			t.Errorf("%s: %+v has an empty item", language, l)
		}
	}

	if tray.LabelsFor("ru").Exit == tray.LabelsFor("en").Exit {
		t.Error("both languages word the menu alike")
	}
	if tray.LabelsFor("de") != tray.LabelsFor("en") {
		t.Error("another language does not get English")
	}
}

func TestTheIconIsTheCrosshairInsideItsSquare(t *testing.T) {
	for _, side := range []int{16, 20, 24, 32, 40, 48, 64} {
		pixels, err := tray.Icon(side)
		if err != nil {
			t.Fatal(err)
		}
		if len(pixels) != 4*side*side {
			t.Fatalf("side %d: %d bytes, want %d", side, len(pixels), 4*side*side)
		}

		at := func(x, y int) []byte { return pixels[4*(y*side+x):][:4] }
		for i := 0; i < side; i++ {
			for _, edge := range [][]byte{at(i, 0), at(i, side-1), at(0, i), at(side-1, i)} {
				if edge[3] != 0 {
					t.Errorf("side %d: the crosshair reaches the edge of the icon", side)
				}
			}
		}

		arm := at(side/2+side/8+side/8, side/2)
		if arm[0] != 0x66 || arm[1] != 0xFF || arm[2] != 0x00 || arm[3] != 0xFF {
			t.Errorf("side %d: an arm pixel is %v, want the default green as blue, green, red, alpha", side, arm)
		}
	}
}

func TestWindowsStartsTheProgramByItsWholePath(t *testing.T) {
	if got, want := tray.StartCommand(`C:\Program Files\NormPrichel\normprichel.exe`), `"C:\Program Files\NormPrichel\normprichel.exe"`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
