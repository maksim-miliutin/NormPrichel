//go:build integration

package profiles_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

func TestTheRealDiskReplacesSettingsWholesale(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "NormPrichel")
	store := profiles.Store{Disk: profiles.OS{}, Dir: dir}

	first := profiles.Defaults()
	first.Visible = false
	second := profiles.Defaults()
	second.Hotkeys.Toggle = "Ctrl+Alt+F9"

	for _, settings := range []profiles.Settings{first, second} {
		if err := store.Save(settings); err != nil {
			t.Fatal(err)
		}
	}

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, second) {
		t.Errorf("got %+v, want the second save", got)
	}

	names, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0].Name() != "profiles.json" {
		t.Errorf("folder holds %v, want profiles.json alone", names)
	}
}

func TestTheRealDiskKeepsOnlyTheLatestBrokenFile(t *testing.T) {
	dir := t.TempDir()
	store := profiles.Store{Disk: profiles.OS{}, Dir: dir}

	for _, broken := range []string{`{ "version": 1, `, `{ "version": 2 }`} {
		if err := os.WriteFile(filepath.Join(dir, "profiles.json"), []byte(broken), 0o644); err != nil {
			t.Fatal(err)
		}

		got, err := store.Load()
		if !errors.Is(err, profiles.ErrBroken) || !reflect.DeepEqual(got, profiles.Defaults()) {
			t.Fatalf("got %+v, %v; want the defaults and ErrBroken", got, err)
		}

		aside, err := os.ReadFile(filepath.Join(dir, "profiles.json.bad"))
		if err != nil || string(aside) != broken {
			t.Errorf("profiles.json.bad holds %q, %v; want %q", aside, err, broken)
		}
		if _, err := os.Stat(filepath.Join(dir, "profiles.json")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("the broken profiles.json is still in place: %v", err)
		}
	}
}

func TestSettingsLiveInAppDataOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the folder is named after %APPDATA%, which only Windows has")
	}

	got, err := profiles.DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(os.Getenv("APPDATA"), "NormPrichel"); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
