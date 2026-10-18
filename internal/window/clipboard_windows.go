package window

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var ErrClipboard = errors.New("window: the clipboard is out of reach")

var (
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procGetClipboardData = user32.NewProc("GetClipboardData")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalFree       = kernel32.NewProc("GlobalFree")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

// Memory that SetClipboardData takes belongs to Windows from then on; only memory it refused is
// freed here.
func (w *Window) Copy(text string) error {
	words, err := windows.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrClipboard, err)
	}
	memory, _, err := procGlobalAlloc.Call(gmemMoveable, uintptr(len(words)*2))
	if memory == 0 {
		return fmt.Errorf("%w: %w", ErrClipboard, err)
	}
	locked, _, err := procGlobalLock.Call(memory)
	if locked == 0 {
		procGlobalFree.Call(memory)
		return fmt.Errorf("%w: %w", ErrClipboard, err)
	}
	copy(unsafe.Slice(at[uint16](locked), len(words)), words)
	procGlobalUnlock.Call(memory)

	if opened, _, err := procOpenClipboard.Call(w.hwnd); opened == 0 {
		procGlobalFree.Call(memory)
		return fmt.Errorf("%w: %w", ErrClipboard, err)
	}
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()
	if set, _, err := procSetClipboardData.Call(cfUnicodeText, memory); set == 0 {
		procGlobalFree.Call(memory)
		return fmt.Errorf("%w: %w", ErrClipboard, err)
	}

	return nil
}

func (w *Window) Paste() (string, error) {
	if opened, _, err := procOpenClipboard.Call(w.hwnd); opened == 0 {
		return "", fmt.Errorf("%w: %w", ErrClipboard, err)
	}
	defer procCloseClipboard.Call()

	handle, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if handle == 0 {
		return "", nil
	}
	locked, _, _ := procGlobalLock.Call(handle)
	if locked == 0 {
		return "", nil
	}
	defer procGlobalUnlock.Call(handle)

	return windows.UTF16PtrToString(at[uint16](locked)), nil
}
