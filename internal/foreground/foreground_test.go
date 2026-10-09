package foreground_test

import (
	"errors"
	"image"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

const own = 4242

var errDenied = errors.New("access is denied")

// desktop stands in for Windows: a window in front, a process list, and a count of snapshots.
type desktop struct {
	window    foreground.Window
	processes []foreground.Process
	fail      error
	snapshots int
}

func (d *desktop) Foreground() foreground.Window {
	return d.window
}

func (d *desktop) Processes() ([]foreground.Process, error) {
	d.snapshots++
	if d.fail != nil {
		return nil, d.fail
	}

	return d.processes, nil
}

func listed() profiles.Settings {
	settings := profiles.Defaults()
	settings.Games = []profiles.Game{
		{Exe: "stalzone.exe", Name: "STALZONE", Enabled: true},
		{Exe: "paused.exe", Name: "Paused", Enabled: false},
		{Exe: "normprichel.exe", Name: "Itself", Enabled: true},
	}

	return settings
}

var processes = []foreground.Process{
	{ID: 7, Exe: "explorer.exe"},
	{ID: 42, Exe: "STALZONE.EXE"},
	{ID: 43, Exe: "paused.exe"},
	{ID: 44, Exe: "notepad.exe"},
	{ID: own, Exe: "normprichel.exe"},
}

var client = image.Rect(-1920, 40, -320, 940)

func TestTheGameInFrontIsFoundByItsExe(t *testing.T) {
	d := &desktop{window: foreground.Window{Handle: 10, Process: 42, Client: client}, processes: processes}

	target, ok, err := foreground.NewWatcher(d, own).Target(listed())
	if err != nil {
		t.Fatal(err)
	}

	if !ok || target.Game.Name != "STALZONE" || target.Client != client {
		t.Errorf("got %+v, %v; want the STALZONE profile over %v", target, ok, client)
	}
}

func TestNothingShows(t *testing.T) {
	hidden := listed()
	hidden.Visible = false

	cases := []struct {
		when     string
		window   foreground.Window
		settings profiles.Settings
	}{
		{when: "while Windows is between two windows", window: foreground.Window{}, settings: listed()},
		{when: "over a window of this very program", window: foreground.Window{Handle: 11, Process: own}, settings: listed()},
		{when: "while the game is minimised", window: foreground.Window{Handle: 10, Process: 42, Minimized: true}, settings: listed()},
		{when: "over a game switched off for now", window: foreground.Window{Handle: 12, Process: 43}, settings: listed()},
		{when: "while the crosshair is hidden everywhere", window: foreground.Window{Handle: 10, Process: 42}, settings: hidden},
		{when: "over a program off the list", window: foreground.Window{Handle: 13, Process: 44}, settings: listed()},
		{when: "over a window whose process has exited", window: foreground.Window{Handle: 14, Process: 99}, settings: listed()},
	}

	for _, c := range cases {
		d := &desktop{window: c.window, processes: processes}

		target, ok, err := foreground.NewWatcher(d, own).Target(c.settings)
		if err != nil {
			t.Fatalf("%s: %v", c.when, err)
		}
		if ok {
			t.Errorf("%s: got %+v, want nothing", c.when, target)
		}
	}
}

func TestSwitchingWindowsTakesNoSnapshot(t *testing.T) {
	d := &desktop{window: foreground.Window{Handle: 10, Process: 42}, processes: processes}
	watcher := foreground.NewWatcher(d, own)
	if _, _, err := watcher.Target(listed()); err != nil {
		t.Fatal(err)
	}

	d.window = foreground.Window{}
	if _, _, err := watcher.Target(listed()); err != nil {
		t.Fatal(err)
	}
	if d.snapshots != 1 {
		t.Errorf("took %d snapshots, want only the one for the game", d.snapshots)
	}
}

func TestTheProcessListIsReadOnlyWhenTheWindowChanges(t *testing.T) {
	d := &desktop{processes: processes}
	watcher := foreground.NewWatcher(d, own)

	steps := []struct {
		window    foreground.Window
		snapshots int
	}{
		{window: foreground.Window{Handle: 10, Process: 42}, snapshots: 1},
		{window: foreground.Window{Handle: 10, Process: 42}, snapshots: 1},
		{window: foreground.Window{Handle: 10, Process: 42, Client: client}, snapshots: 1},
		{window: foreground.Window{Handle: 10, Process: 42, Minimized: true}, snapshots: 1},
		{window: foreground.Window{Handle: 20, Process: 42}, snapshots: 2},
		{window: foreground.Window{Handle: 20, Process: 44}, snapshots: 3},
		{window: foreground.Window{Handle: 20, Process: 44}, snapshots: 3},
	}

	for i, step := range steps {
		d.window = step.window
		if _, _, err := watcher.Target(listed()); err != nil {
			t.Fatal(err)
		}
		if d.snapshots != step.snapshots {
			t.Errorf("step %d: %d snapshots, want %d", i, d.snapshots, step.snapshots)
		}
	}
}

func TestAMovedWindowIsReportedWhereItIsNow(t *testing.T) {
	d := &desktop{window: foreground.Window{Handle: 10, Process: 42, Client: client}, processes: processes}
	watcher := foreground.NewWatcher(d, own)
	if _, _, err := watcher.Target(listed()); err != nil {
		t.Fatal(err)
	}

	moved := client.Add(image.Pt(1920, 0))
	d.window.Client = moved
	target, ok, err := watcher.Target(listed())
	if err != nil || !ok || target.Client != moved {
		t.Errorf("got %+v, %v, %v; want the game over %v", target, ok, err, moved)
	}
}

func TestAWindowOfAnExitedProcessDoesNotInheritTheLastGame(t *testing.T) {
	d := &desktop{window: foreground.Window{Handle: 10, Process: 42}, processes: processes}
	watcher := foreground.NewWatcher(d, own)
	if _, ok, err := watcher.Target(listed()); err != nil || !ok {
		t.Fatalf("got %v, %v; want STALZONE first", ok, err)
	}

	d.window = foreground.Window{Handle: 30, Process: 99}
	if target, ok, err := watcher.Target(listed()); err != nil || ok {
		t.Errorf("got %+v, %v, %v; want nothing for a process that is gone", target, ok, err)
	}
}

func TestAFailedSnapshotIsTakenAgain(t *testing.T) {
	d := &desktop{window: foreground.Window{Handle: 10, Process: 42}, processes: processes, fail: errDenied}
	watcher := foreground.NewWatcher(d, own)

	if _, ok, err := watcher.Target(listed()); !errors.Is(err, errDenied) || ok {
		t.Fatalf("got %v, %v; want the snapshot error and no game", ok, err)
	}

	d.fail = nil
	target, ok, err := watcher.Target(listed())
	if err != nil || !ok || target.Game.Name != "STALZONE" {
		t.Errorf("got %+v, %v, %v; want STALZONE once the snapshot works", target, ok, err)
	}
	if d.snapshots != 2 {
		t.Errorf("%d snapshots, want a second one after the failure", d.snapshots)
	}
}
