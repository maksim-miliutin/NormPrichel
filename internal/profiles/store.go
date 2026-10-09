package profiles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

var (
	ErrUnreadable = errors.New("profiles: the settings file cannot be read")
	ErrNotSaved   = errors.New("profiles: the settings were not saved")
)

const fileName = "profiles.json"

type Disk interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte) error
	Rename(from, to string) error
	Remove(name string) error
	MkdirAll(name string) error
}

type Store struct {
	Disk Disk
	Dir  string
}

func DefaultDir() (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(config, "NormPrichel"), nil
}

// Load hands back usable settings even when it fails: the defaults, with the reason in the error.
func (s Store) Load() (Settings, error) {
	name := filepath.Join(s.Dir, fileName)
	text, err := s.Disk.ReadFile(name)
	if errors.Is(err, fs.ErrNotExist) {
		return Defaults(), nil
	}
	if err != nil {
		return Defaults(), fmt.Errorf("%w: %w", ErrUnreadable, err)
	}

	settings, err := Parse(text)
	if err == nil {
		return settings, nil
	}

	return Defaults(), errors.Join(err, s.Disk.Rename(name, name+".bad"))
}

func (s Store) Save(settings Settings) error {
	text, err := settings.Encode()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNotSaved, err)
	}
	if err := s.Disk.MkdirAll(s.Dir); err != nil {
		return fmt.Errorf("%w: %w", ErrNotSaved, err)
	}

	name := filepath.Join(s.Dir, fileName)
	if err := s.replace(name, text); err != nil {
		return fmt.Errorf("%w: %w", ErrNotSaved, errors.Join(err, s.discard(name+".tmp")))
	}

	return nil
}

func (s Store) discard(name string) error {
	if err := s.Disk.Remove(name); !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	return nil
}

func (s Store) replace(name string, text []byte) error {
	if err := s.Disk.WriteFile(name+".tmp", text); err != nil {
		return err
	}

	return s.Disk.Rename(name+".tmp", name)
}

type OS struct{}

func (OS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// Sync before close: renaming over the old file must not reach the disk before the new bytes do.
func (OS) WriteFile(name string, data []byte) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err == nil {
		err = file.Sync()
	}

	return errors.Join(err, file.Close())
}

func (OS) Rename(from, to string) error {
	return os.Rename(from, to)
}

func (OS) Remove(name string) error {
	return os.Remove(name)
}

func (OS) MkdirAll(name string) error {
	return os.MkdirAll(name, 0o755)
}
