//go:build integration

package tray_test

import (
	"errors"
	"os"
	"testing"

	"golang.org/x/sys/windows/registry"

	"github.com/maksim-miliutin/NormPrichel/internal/tray"
)

// The test switches the real entry, so it puts back whatever the player had there before.
func TestAutostartIsSwitchedOnForThisFileAndOffAgain(t *testing.T) {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer key.Close()
	before, _, err := key.GetStringValue("NormPrichel")
	had := err == nil
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		t.Fatal(err)
	}
	defer func() {
		if had {
			key.SetStringValue("NormPrichel", before)
			return
		}
		key.DeleteValue("NormPrichel")
	}()

	if err := tray.SetAutostart(true); err != nil {
		t.Fatal(err)
	}
	exe, _ := os.Executable()
	if on, err := tray.Autostart(); err != nil || !on {
		t.Errorf("after switching on: %v, %v; want on", on, err)
	}
	if value, _, _ := key.GetStringValue("NormPrichel"); value != tray.StartCommand(exe) {
		t.Errorf("Windows would run %s, want %s", value, tray.StartCommand(exe))
	}

	if err := tray.SetAutostart(false); err != nil {
		t.Fatal(err)
	}
	if on, err := tray.Autostart(); err != nil || on {
		t.Errorf("after switching off: %v, %v; want off", on, err)
	}
	if err := tray.SetAutostart(false); err != nil {
		t.Errorf("switching off twice: %v, want no error", err)
	}
}
