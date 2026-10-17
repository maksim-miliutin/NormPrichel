package live_test

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/live"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var errFull = errors.New("the disk is full")

// disk records every save; with a gate, each save waits until the test lets it through, and
// started tells the test that a save has begun.
type disk struct {
	mu      sync.Mutex
	saves   []profiles.Settings
	gate    chan struct{}
	started chan struct{}
	fail    error
}

func (d *disk) Save(settings profiles.Settings) error {
	if d.started != nil {
		d.started <- struct{}{}
	}
	if d.gate != nil {
		<-d.gate
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.saves = append(d.saves, settings)

	return d.fail
}

func (d *disk) saved() []profiles.Settings {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.saves
}

// journal collects what the state reports from its saving goroutine.
type journal struct {
	mu      sync.Mutex
	entries []error
}

func (j *journal) report(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.entries = append(j.entries, err)
}

func (j *journal) read() []error {
	j.mu.Lock()
	defer j.mu.Unlock()

	return j.entries
}

func ignore(error) {}

func await(t *testing.T, started <-chan struct{}) {
	t.Helper()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("no save began within 5 s")
	}
}

func crosshairOf(shape crosshair.Shape) profiles.Crosshair {
	c := profiles.DefaultCrosshair()
	c.Shape = shape

	return c
}

func listed() profiles.Settings {
	settings := profiles.Defaults()
	settings.Games = []profiles.Game{
		{Exe: "stalzone.exe", Name: "STALZONE", Enabled: true, Crosshair: crosshairOf(crosshair.Dot)},
		{Exe: "other.exe", Name: "Other", Enabled: true, Crosshair: crosshairOf(crosshair.Cross)},
	}

	return settings
}

func set(change func(*profiles.Settings)) func(profiles.Settings) (profiles.Settings, error) {
	return func(s profiles.Settings) (profiles.Settings, error) {
		change(&s)
		return s, nil
	}
}

func renamed(name string) func(profiles.Settings) (profiles.Settings, error) {
	return func(s profiles.Settings) (profiles.Settings, error) {
		return s.Edited("stalzone.exe", func(g profiles.Game) profiles.Game {
			g.Name = name
			return g
		})
	}
}

func TestAChangeIsKeptAsTheDiskWouldGiveItBack(t *testing.T) {
	state := live.New(listed(), &disk{}, ignore)
	defer state.Close()

	got, err := state.Change(set(func(s *profiles.Settings) {
		s.Games = append(s.Games, profiles.Game{Exe: "STALZONE.EXE", Name: "twice"})
		s.Games[0].Offset.X = 900
	}))
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Games) != 2 || got.Games[0].Offset.X != 500 {
		t.Errorf("got games %+v, want the copy dropped and the offset clamped to 500", got.Games)
	}
	if current := state.Current(); current.Games[0].Offset.X != 500 || len(current.Games) != 2 {
		t.Errorf("current games %+v, want what the change returned", current.Games)
	}
}

func TestARefusedChangeLeavesTheSettingsAndTheDiskAlone(t *testing.T) {
	d := &disk{}
	state := live.New(listed(), d, ignore)

	_, err := state.Change(set(func(s *profiles.Settings) { s.Games[0].Crosshair.Shape = "star" }))
	if !errors.Is(err, live.ErrRefused) || !errors.Is(err, profiles.ErrBroken) {
		t.Errorf("an unknown shape: got %v, want ErrRefused wrapping the reason", err)
	}

	_, err = state.Change(func(s profiles.Settings) (profiles.Settings, error) { return s.WithoutGame("notepad.exe") })
	if !errors.Is(err, profiles.ErrNoGame) {
		t.Errorf("a failing edit: got %v, want its own error", err)
	}

	state.Close()
	if current := state.Current(); current.Games[0].Crosshair.Shape != crosshair.Dot || len(d.saved()) != 0 {
		t.Errorf("crosshair %q and %d saves, want the dot and nothing saved", current.Games[0].Crosshair.Shape, len(d.saved()))
	}
}

func TestChangesThatComeWhileTheDiskIsBusyFoldIntoOneSave(t *testing.T) {
	d := &disk{gate: make(chan struct{}), started: make(chan struct{})}
	state := live.New(listed(), d, ignore)

	state.Change(renamed("a"))
	await(t, d.started)
	for _, name := range []string{"b", "c", "d"} {
		state.Change(renamed(name))
	}
	d.gate <- struct{}{}

	await(t, d.started)
	d.gate <- struct{}{}
	go func() {
		for range d.started {
			d.gate <- struct{}{}
		}
	}()
	state.Close()
	close(d.started)

	saves := d.saved()
	if len(saves) != 2 || saves[0].Games[0].Name != "a" || saves[1].Games[0].Name != "d" {
		t.Errorf("saved %d times, want twice: a, then d alone", len(saves))
	}
}

func TestClosingWaitsForTheLastChangeToReachTheDisk(t *testing.T) {
	d := &disk{gate: make(chan struct{})}
	state := live.New(listed(), d, ignore)
	state.Change(renamed("last"))

	closed := make(chan struct{})
	go func() {
		state.Close()
		close(closed)
	}()
	select {
	case <-closed:
		t.Fatal("Close returned while the save was still waiting for the disk")
	case <-time.After(50 * time.Millisecond):
	}

	d.gate <- struct{}{}
	<-closed
	if saves := d.saved(); len(saves) != 1 || saves[0].Games[0].Name != "last" {
		t.Errorf("saved %d times, want the last change once", len(saves))
	}
}

func TestAChangeAfterCloseIsRefused(t *testing.T) {
	state := live.New(listed(), &disk{}, ignore)
	state.Close()

	if _, err := state.Change(renamed("late")); !errors.Is(err, live.ErrClosed) {
		t.Errorf("got %v, want ErrClosed", err)
	}
}

// A save that works is reported too, as nil: the reporter prints a lasting fault once, and
// needs to hear when it ended to print it again the next time.
func TestEverySaveIsReportedFailedOrNot(t *testing.T) {
	for _, fail := range []error{errFull, nil} {
		j := &journal{}
		state := live.New(listed(), &disk{fail: fail}, j.report)

		state.Change(renamed("first"))
		state.Close()

		if entries := j.read(); len(entries) != 1 || entries[0] != fail {
			t.Errorf("a save that returns %v: reported %v, want exactly that", fail, entries)
		}
	}
}

func TestANewGameIsOfferedTheCrosshairChosenLast(t *testing.T) {
	state := live.New(listed(), &disk{}, ignore)
	defer state.Close()

	if got := state.Last(); got.Shape != crosshair.Cross {
		t.Errorf("at the start: %q, want the cross of the last game listed", got.Shape)
	}

	state.Change(set(func(s *profiles.Settings) { s.Games[0].Offset.X = 3 }))
	if got := state.Last(); got.Shape != crosshair.Cross {
		t.Errorf("after a nudge: %q, want the cross still", got.Shape)
	}

	state.Change(set(func(s *profiles.Settings) { s.Games[0].Crosshair = crosshairOf(crosshair.Circle) }))
	if got := state.Last(); got.Shape != crosshair.Circle {
		t.Errorf("after a new crosshair: %q, want the circle", got.Shape)
	}

	state.Change(set(func(s *profiles.Settings) {
		s.Games = append(s.Games, profiles.Game{Exe: "new.exe", Crosshair: crosshairOf(crosshair.Tee)})
	}))
	if got := state.Last(); got.Shape != crosshair.Circle {
		t.Errorf("after a game was added: %q, want the circle still", got.Shape)
	}
}

func TestWithNoGamesTheDefaultCrosshairIsOffered(t *testing.T) {
	state := live.New(profiles.Defaults(), &disk{}, ignore)
	defer state.Close()

	if got := state.Last(); got != profiles.DefaultCrosshair() {
		t.Errorf("got %+v, want the default crosshair", got)
	}
}

func TestAResetBringsBackAFirstStartAndForgetsTheCrosshairChosenLast(t *testing.T) {
	d := &disk{}
	state := live.New(listed(), d, ignore)
	state.Change(set(func(s *profiles.Settings) { s.Games[0].Crosshair = crosshairOf(crosshair.Circle) }))

	got, err := state.Reset()
	state.Close()

	if err != nil || !reflect.DeepEqual(got, profiles.Defaults()) {
		t.Errorf("the reset gives %+v, %v; want the settings of a first start", got, err)
	}
	if last := state.Last(); last != profiles.DefaultCrosshair() {
		t.Errorf("a game added next would start from %+v, want the default crosshair", last)
	}
	if saves := d.saved(); len(saves) == 0 || !reflect.DeepEqual(saves[len(saves)-1], profiles.Defaults()) {
		t.Errorf("the disk keeps %+v, want the settings of a first start", saves)
	}
	if _, err := state.Reset(); !errors.Is(err, live.ErrClosed) {
		t.Errorf("a reset after close gives %v, want ErrClosed", err)
	}
}

func TestTheSettingsHandedOutAreACopy(t *testing.T) {
	state := live.New(listed(), &disk{}, ignore)
	defer state.Close()

	current := state.Current()
	current.Games[0].Name = "scribbled"

	state.Change(func(s profiles.Settings) (profiles.Settings, error) {
		s.Games[1].Name = "scribbled too"
		return s, errFull
	})

	if got := state.Current(); got.Games[0].Name != "STALZONE" || got.Games[1].Name != "Other" {
		t.Errorf("games %+v, want them untouched by writes to copies", got.Games)
	}
}

func TestChangesFromManyGoroutinesAreNotLost(t *testing.T) {
	state := live.New(listed(), &disk{}, ignore)

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 50 {
				state.Change(set(func(s *profiles.Settings) { s.Games[0].Offset.X++ }))
				state.Current()
			}
		})
	}
	wg.Wait()
	state.Close()

	if got := state.Current().Games[0].Offset.X; got != 400 {
		t.Errorf("offset %d after 400 nudges of one pixel, want 400", got)
	}
}
