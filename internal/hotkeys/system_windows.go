package hotkeys

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"golang.org/x/sys/windows"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
)

// System registers hotkeys for one window. Windows lets only the thread that made the window
// register them, and hands their presses to that window as WM_HOTKEY.
type System struct {
	Window uintptr
}

func (s System) Register(b Binding) error {
	ok, _, err := procRegisterHotKey.Call(s.Window, uintptr(b.Action), uintptr(b.Flags()), uintptr(b.Key))
	if ok != 0 {
		return nil
	}
	if errors.Is(err, windows.ERROR_HOTKEY_ALREADY_REGISTERED) {
		return fmt.Errorf("%w: %w", ErrTaken, err)
	}

	return err
}

func (s System) Unregister(b Binding) error {
	if ok, _, err := procUnregisterHotKey.Call(s.Window, uintptr(b.Action)); ok == 0 {
		return err
	}

	return nil
}

// Two probes at once would hold the same hotkey on two threads, and each would take the other's
// hold for another program's; they take turns instead.
var probing sync.Mutex

// Taken holds each binding for a moment and lets it go: a refusal means another program has it.
// Asked while the game is in front, it would find this program's own hotkeys too.
func Taken(bindings []Binding) []Binding {
	probing.Lock()
	defer probing.Unlock()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var taken []Binding
	for _, b := range bindings {
		if err := (System{}).Register(b); err != nil {
			taken = append(taken, b)
			continue
		}
		(System{}).Unregister(b)
	}

	return taken
}
