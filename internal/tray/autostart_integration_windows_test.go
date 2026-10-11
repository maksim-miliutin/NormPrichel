//go:build integration

package tray

import (
	"os"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestAutostartIsSwitchedForThisFileAndRespectsTaskManager(t *testing.T) {
	autostartName = "NormPrichel integration test"
	defer func() { autostartName = "NormPrichel" }()
	defer clean()

	if err := SetAutostart(true); err != nil {
		t.Fatal(err)
	}
	exe, _ := os.Executable()
	if on, err := Autostart(); err != nil || !on {
		t.Errorf("after switching on: %v, %v; want on", on, err)
	}
	if got := read(t, runKey); got != StartCommand(exe) {
		t.Errorf("Windows would run %s, want %s", got, StartCommand(exe))
	}

	mark(t, []byte{0x03, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	if on, err := Autostart(); err != nil || on {
		t.Errorf("switched off in Task Manager: %v, %v; want off", on, err)
	}
	if err := SetAutostart(true); err != nil {
		t.Fatal(err)
	}
	if on, err := Autostart(); err != nil || !on {
		t.Errorf("switched on again from the menu: %v, %v; want on", on, err)
	}

	if err := SetAutostart(false); err != nil {
		t.Fatal(err)
	}
	if on, err := Autostart(); err != nil || on {
		t.Errorf("after switching off: %v, %v; want off", on, err)
	}
	if err := SetAutostart(false); err != nil {
		t.Errorf("switching off twice: %v, want no error", err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()

	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer key.Close()
	value, _, _ := key.GetStringValue(autostartName)

	return value
}

func mark(t *testing.T, value []byte) {
	t.Helper()

	key, _, err := registry.CreateKey(registry.CURRENT_USER, approvedKey, registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer key.Close()
	if err := key.SetBinaryValue(autostartName, value); err != nil {
		t.Fatal(err)
	}
}

func clean() {
	for _, path := range []string{runKey, approvedKey} {
		if key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.SET_VALUE); err == nil {
			key.DeleteValue(autostartName)
			key.Close()
		}
	}
}
