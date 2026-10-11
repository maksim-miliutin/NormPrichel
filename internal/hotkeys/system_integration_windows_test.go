//go:build integration

package hotkeys_test

import (
	"errors"
	"runtime"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

// F24 is on no keyboard, so no program is likely to hold it.
func rare(t *testing.T) hotkeys.Binding {
	t.Helper()

	keys := profiles.Defaults().Hotkeys
	keys.Toggle = "Ctrl+Shift+Alt+F24"
	bindings, err := hotkeys.Bindings(keys)
	if err != nil {
		t.Fatal(err)
	}

	return bindings[0]
}

func TestAHeldHotkeyIsTakenForEveryoneElseUntilLetGo(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	b := rare(t)

	if taken := hotkeys.Taken([]hotkeys.Binding{b}); len(taken) != 0 {
		t.Fatalf("%s is taken before the test holds it: close the program that has it", b.Text)
	}

	if err := (hotkeys.System{}).Register(b); err != nil {
		t.Fatal(err)
	}
	if err := (hotkeys.System{}).Register(b); !errors.Is(err, hotkeys.ErrTaken) {
		t.Errorf("registering it twice: got %v, want ErrTaken", err)
	}
	if taken := hotkeys.Taken([]hotkeys.Binding{b}); len(taken) != 1 {
		t.Errorf("while held, %s is not seen as taken", b.Text)
	}

	if err := (hotkeys.System{}).Unregister(b); err != nil {
		t.Fatal(err)
	}
	if taken := hotkeys.Taken([]hotkeys.Binding{b}); len(taken) != 0 {
		t.Errorf("after letting go, %s is still seen as taken", b.Text)
	}
}
