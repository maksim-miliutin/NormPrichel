package tray

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

var ErrAutostart = errors.New("tray: starting with Windows cannot be switched")

const (
	runKey    = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName = "NormPrichel"
)

// Autostart is on only when Windows would start this very file: after the file moves, the old
// entry points nowhere and the menu shows it off.
func Autostart() (bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrAutostart, err)
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrAutostart, err)
	}
	defer key.Close()

	value, _, err := key.GetStringValue(valueName)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrAutostart, err)
	}

	return value == StartCommand(exe), nil
}

func SetAutostart(on bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAutostart, err)
	}
	defer key.Close()

	if !on {
		if err := key.DeleteValue(valueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("%w: %w", ErrAutostart, err)
		}
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAutostart, err)
	}
	if err := key.SetStringValue(valueName, StartCommand(exe)); err != nil {
		return fmt.Errorf("%w: %w", ErrAutostart, err)
	}

	return nil
}
