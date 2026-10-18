package window_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"slices"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/live"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
	"github.com/maksim-miliutin/NormPrichel/internal/settings"
	"github.com/maksim-miliutin/NormPrichel/internal/window"
)

// disk keeps files in a map, enough for the picture store behind the settings.
type disk map[string][]byte

func (d disk) ReadFile(name string) ([]byte, error) {
	file, ok := d[name]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return file, nil
}

func (d disk) WriteFile(name string, contents []byte) error {
	d[name] = contents
	return nil
}

func (d disk) Rename(from, to string) error {
	d[to] = d[from]
	delete(d, from)
	return nil
}

func (d disk) Remove(name string) error {
	delete(d, name)
	return nil
}

func (disk) MkdirAll(string) error { return nil }

// changes counts the changes the form asks for, so a field left as it was can be seen to write nothing.
type changes struct {
	settings.State
	count int
}

func (c *changes) Change(edit func(profiles.Settings) (profiles.Settings, error)) (profiles.Settings, error) {
	c.count++
	return c.State.Change(edit)
}

type nowhere struct{}

func (nowhere) Save(profiles.Settings) error { return nil }

// machine stands for what the form asks of Windows.
type machine struct {
	windows []foreground.Choice
	taken   []string
	starts  bool
}

func (m *machine) desktop() ([]foreground.Choice, error) {
	return m.windows, nil
}

func (m *machine) held(bindings []hotkeys.Binding) []hotkeys.Binding {
	return slices.DeleteFunc(slices.Clone(bindings), func(b hotkeys.Binding) bool { return !slices.Contains(m.taken, b.Text) })
}

func (m *machine) autostart() (bool, error) {
	return m.starts, nil
}

func (m *machine) switchAutostart(on bool) error {
	m.starts = on
	return nil
}

// player answers for the window: what the fields hold and what the dialogs give back.
type player struct {
	texts        map[int]string
	rows         map[int]int
	agrees       bool
	asked        []string
	colour       crosshair.RGB
	colourChosen bool
	started      crosshair.RGB
	picture      io.ReadCloser
	offered      []string
	none         string
	pick         int
	picked       bool
	visited      []string
	copied       []string
	clipboard    string
}

func (p *player) Text(id int) string { return p.texts[id] }

func (p *player) Row(id int) int {
	if row, ok := p.rows[id]; ok {
		return row
	}

	return -1
}

func (p *player) Confirm(question string) bool {
	p.asked = append(p.asked, question)
	return p.agrees
}

func (p *player) Colour(start crosshair.RGB) (crosshair.RGB, bool) {
	p.started = start
	return p.colour, p.colourChosen
}

func (p *player) Picture() (io.ReadCloser, error) {
	return p.picture, nil
}

func (p *player) Pick(rows []string, none string) (int, bool) {
	p.offered, p.none = rows, none
	return p.pick, p.picked
}

func (p *player) Copy(text string) error {
	p.copied = append(p.copied, text)
	return nil
}

func (p *player) Paste() (string, error) {
	return p.clipboard, nil
}

func (p *player) Visit(address string) error {
	p.visited = append(p.visited, address)
	return nil
}

// counted hands out a file of endless zeros and counts how much of it was read.
type counted struct {
	read   int
	closed bool
}

func (c *counted) Read(b []byte) (int, error) {
	clear(b)
	c.read += len(b)
	return len(b), nil
}

func (c *counted) Close() error {
	c.closed = true
	return nil
}

func listed() profiles.Settings {
	s := profiles.Defaults()
	for _, game := range []profiles.Game{
		{Exe: "stalzone.exe", Name: "STALZONE", Enabled: true},
		{Exe: "other.exe", Name: "Other", Enabled: true},
	} {
		game.Crosshair = profiles.DefaultCrosshair()
		s.Games = append(s.Games, game)
	}

	return s
}

func newActions(t *testing.T, s profiles.Settings) (window.Actions, *changes, *machine, *player) {
	t.Helper()

	inner := live.New(s, nowhere{}, func(error) {})
	t.Cleanup(inner.Close)
	state := &changes{State: inner}
	m := &machine{}
	p := &player{texts: map[int]string{}, rows: map[int]int{}}
	form := &settings.Form{
		State:           state,
		Pictures:        profiles.Store{Disk: disk{}, Dir: "NormPrichel"},
		Windows:         m.desktop,
		Taken:           m.held,
		Autostart:       m.autostart,
		SwitchAutostart: m.switchAutostart,
		Version:         "v0.1.0",
		WindowsLanguage: "ru",
	}

	return window.Actions{Form: form, Inputs: p}, state, m, p
}

// control finds the number of the control that matches, as the window shows the form now.
func control(t *testing.T, a window.Actions, match func(window.Control) bool) int {
	t.Helper()
	controls := window.Controls(a.Form.Screen(), a.Form.Word)
	i := slices.IndexFunc(controls, match)
	if i < 0 {
		t.Fatal("no such control on the screen")
	}

	return controls[i].ID
}

func worded(word string) func(window.Control) bool {
	return func(c window.Control) bool { return c.Word == word }
}

func reading(text string) func(window.Control) bool {
	return func(c window.Control) bool { return c.Text == text }
}

func nth(kind window.Kind, n int) func(window.Control) bool {
	seen := 0
	return func(c window.Control) bool {
		if c.Kind != kind {
			return false
		}
		seen++
		return seen == n+1
	}
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
