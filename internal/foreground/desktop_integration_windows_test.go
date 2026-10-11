//go:build integration

package foreground_test

import (
	"os"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
)

func TestTheSnapshotListsThisVeryTest(t *testing.T) {
	processes, err := foreground.System{}.Processes()
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range processes {
		if p.ID == uint32(os.Getpid()) {
			if !strings.EqualFold(p.Exe, "foreground.test.exe") {
				t.Errorf("this process is listed as %q, want foreground.test.exe", p.Exe)
			}
			return
		}
	}
	t.Errorf("process %d is missing from %d listed", os.Getpid(), len(processes))
}

func TestTheWindowInFrontBelongsToAListedProcess(t *testing.T) {
	window := foreground.System{}.Foreground()
	if window.Handle == 0 {
		t.Skip("no window in front: run the test from a desktop session, not a service")
	}
	if !window.Minimized && window.Client.Empty() {
		t.Errorf("the window in front has an empty client area %v", window.Client)
	}

	processes, err := foreground.System{}.Processes()
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range processes {
		if p.ID == window.Process {
			t.Logf("in front: %s, process %d, client area %v", p.Exe, p.ID, window.Client)
			return
		}
	}
	t.Errorf("the window in front belongs to process %d, which the snapshot does not list", window.Process)
}

func TestTheWindowsToChooseFromAreNamedAndExcludeThisTest(t *testing.T) {
	choices, err := foreground.System{}.Choices(uint32(os.Getpid()))
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range choices {
		t.Logf("%s  (%s)", c.Title, c.Exe)
		if strings.TrimSpace(c.Title) == "" || c.Exe == "" {
			t.Errorf("a choice without a title or an exe: %+v", c)
		}
		if strings.EqualFold(c.Exe, "foreground.test.exe") {
			t.Errorf("the test itself is offered: %+v", c)
		}
		if c.Title == "Program Manager" && strings.EqualFold(c.Exe, "explorer.exe") {
			t.Errorf("the desktop is offered as a game: %+v", c)
		}
	}
}
