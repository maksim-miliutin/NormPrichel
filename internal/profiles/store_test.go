package profiles_test

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var (
	folder    = filepath.Join("config", "NormPrichel")
	saved     = filepath.Join(folder, "profiles.json")
	setAside  = filepath.Join(folder, "profiles.json.bad")
	errDenied = errors.New("access is denied")
)

// memory is a disk in a map; fail names the operations that should return errDenied, where
// "create" fails before a file appears and "write" leaves half of it behind.
type memory struct {
	files map[string]string
	dirs  map[string]bool
	fail  map[string]bool
}

func newMemory(files map[string]string) *memory {
	return &memory{files: files, dirs: map[string]bool{folder: len(files) > 0}, fail: map[string]bool{}}
}

func (m *memory) ReadFile(name string) ([]byte, error) {
	if m.fail["read"] {
		return nil, errDenied
	}
	text, ok := m.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return []byte(text), nil
}

func (m *memory) WriteFile(name string, data []byte) error {
	if m.fail["create"] {
		return errDenied
	}
	if m.fail["write"] {
		m.files[name] = string(data[:len(data)/2])
		return errDenied
	}
	if !m.dirs[filepath.Dir(name)] {
		return fs.ErrNotExist
	}
	m.files[name] = string(data)

	return nil
}

func (m *memory) Rename(from, to string) error {
	if m.fail["rename"] {
		return errDenied
	}
	text, ok := m.files[from]
	if !ok {
		return fs.ErrNotExist
	}
	delete(m.files, from)
	m.files[to] = text

	return nil
}

func (m *memory) Remove(name string) error {
	if _, ok := m.files[name]; !ok {
		return fs.ErrNotExist
	}
	delete(m.files, name)

	return nil
}

func (m *memory) MkdirAll(name string) error {
	m.dirs[name] = true

	return nil
}

func TestNoFileMeansTheDefaults(t *testing.T) {
	disk := newMemory(map[string]string{})

	got, err := profiles.Store{Disk: disk, Dir: folder}.Load()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, profiles.Defaults()) || len(disk.files) != 0 {
		t.Errorf("got %+v with files %v, want the defaults and an untouched disk", got, disk.files)
	}
}

func TestABrokenFileIsMovedAsideAndTheDefaultsTakeOver(t *testing.T) {
	disk := newMemory(map[string]string{saved: `{ "version": 1, `})

	got, err := profiles.Store{Disk: disk, Dir: folder}.Load()
	if !errors.Is(err, profiles.ErrBroken) {
		t.Fatalf("got %v, want ErrBroken", err)
	}

	if !reflect.DeepEqual(got, profiles.Defaults()) {
		t.Errorf("got %+v, want the defaults", got)
	}
	if want := map[string]string{setAside: `{ "version": 1, `}; !reflect.DeepEqual(disk.files, want) {
		t.Errorf("files %v, want the broken file kept under .bad and nothing else", disk.files)
	}
}

func TestAnUnreadableFileIsLeftWhereItIs(t *testing.T) {
	disk := newMemory(map[string]string{saved: `{}`})
	disk.fail["read"] = true

	got, err := profiles.Store{Disk: disk, Dir: folder}.Load()
	if !errors.Is(err, profiles.ErrUnreadable) || !errors.Is(err, errDenied) {
		t.Fatalf("got %v, want ErrUnreadable wrapping the cause", err)
	}

	if !reflect.DeepEqual(got, profiles.Defaults()) || disk.files[saved] != `{}` || len(disk.files) != 1 {
		t.Errorf("got %+v with files %v, want the defaults and the file left alone", got, disk.files)
	}
}

func TestSavedSettingsLoadBackIntoANewFolder(t *testing.T) {
	disk := newMemory(map[string]string{})
	store := profiles.Store{Disk: disk, Dir: folder}

	want := profiles.Defaults()
	want.Visible = false
	want.Games = []profiles.Game{{Exe: "stalzone.exe", Name: "STALZONE", Enabled: true, Crosshair: profiles.Crosshair{Style: specStyle, Scale: 100}, Offset: profiles.Offset{X: 4, Y: -2}}}

	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) || len(disk.files) != 1 {
		t.Errorf("got %+v with files %v, want what was saved in one file", got, disk.files)
	}
}

func TestAFailedSaveLeavesTheOldSettingsIntact(t *testing.T) {
	for _, failing := range []string{"create", "write", "rename"} {
		old := `{ "visible": false }`
		disk := newMemory(map[string]string{saved: old})
		disk.fail[failing] = true

		err := profiles.Store{Disk: disk, Dir: folder}.Save(profiles.Defaults())
		if !errors.Is(err, profiles.ErrNotSaved) || !errors.Is(err, errDenied) || errors.Is(err, fs.ErrNotExist) {
			t.Errorf("failing %s: got %v, want ErrNotSaved wrapping the cause and nothing about cleanup", failing, err)
		}

		if want := map[string]string{saved: old}; !reflect.DeepEqual(disk.files, want) {
			t.Errorf("failing %s: files %v, want the old file alone and no leftovers", failing, disk.files)
		}
	}
}
