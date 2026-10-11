package settings_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"sync"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

// disk keeps files in a map, enough for the picture store behind the settings.
type disk struct {
	mu    sync.Mutex
	files map[string][]byte
}

func (d *disk) ReadFile(name string) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	file, ok := d.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return file, nil
}

func (d *disk) WriteFile(name string, data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.files[name] = data

	return nil
}

func (d *disk) Rename(from, to string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.files[to] = d.files[from]
	delete(d.files, from)

	return nil
}

func (d *disk) Remove(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.files, name)

	return nil
}

func (d *disk) MkdirAll(string) error { return nil }

type nowhere struct{}

func (nowhere) Save(profiles.Settings) error { return nil }

func crosshairOf(shape crosshair.Shape) profiles.Crosshair {
	c := profiles.DefaultCrosshair()
	c.Shape = shape

	return c
}

func listed() profiles.Settings {
	s := profiles.Defaults()
	s.Games = []profiles.Game{
		{Exe: "stalzone.exe", Name: "STALZONE", Enabled: true, Crosshair: crosshairOf(crosshair.Dot)},
		{Exe: "other.exe", Name: "Other", Enabled: true, Crosshair: crosshairOf(crosshair.Circle)},
	}

	return s
}

func equal(a, b profiles.Settings) bool {
	x, _ := a.Encode()
	y, _ := b.Encode()

	return bytes.Equal(x, y)
}

func pngOf(t *testing.T, side int) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, side, side))
	img.SetNRGBA(side/2, side/2, color.NRGBA{R: 255, A: 255})
	var file bytes.Buffer
	if err := png.Encode(&file, img); err != nil {
		t.Fatal(err)
	}

	return file.Bytes()
}
