package tray

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

var ErrAutostart = errors.New("tray: starting with Windows cannot be switched")

const (
	runKey      = `Software\Microsoft\Windows\CurrentVersion\Run`
	approvedKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
)

// The integration test switches the name, so it never touches the player's own entry.
var autostartName = "NormPrichel"

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

	value, _, err := key.GetStringValue(autostartName)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrAutostart, err)
	}

	return value == StartCommand(exe) && approved(), nil
}

func approved() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, approvedKey, registry.QUERY_VALUE)
	if err != nil {
		return true
	}
	defer key.Close()

	mark, _, err := key.GetBinaryValue(autostartName)
	if err != nil {
		return true
	}

	return Approved(mark)
}

// Switching here is the player's latest word, so Task Manager's mark goes, whichever way.
func forgetMark() {
	key, err := registry.OpenKey(registry.CURRENT_USER, approvedKey, registry.SET_VALUE)
	if err != nil {
		return
	}
	defer key.Close()

	key.DeleteValue(autostartName)
}

func SetAutostart(on bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAutostart, err)
	}
	defer key.Close()
	forgetMark()

	if !on {
		if err := key.DeleteValue(autostartName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("%w: %w", ErrAutostart, err)
		}
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAutostart, err)
	}
	if err := key.SetStringValue(autostartName, StartCommand(exe)); err != nil {
		return fmt.Errorf("%w: %w", ErrAutostart, err)
	}

	return nil
}
