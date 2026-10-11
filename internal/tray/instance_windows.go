package tray

import (
	"errors"
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var ErrInstance = errors.New("tray: the copy already running does not answer")

var (
	procFindWindowW              = user32.NewProc("FindWindowW")
	procAllowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
)

const wmWake = 0x8000 + 2

// The claim lasts as long as the process: Windows lets go of the name when the process ends,
// however it ends.
var claim windows.Handle

func Claim() (bool, error) {
	name, _ := windows.UTF16PtrFromString(`Local\NormPrichel`)
	handle, err := windows.CreateMutex(nil, false, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		windows.CloseHandle(handle)
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrInstance, err)
	}
	claim = handle

	return true, nil
}

// Wake asks the running copy to open its page. That copy may still be making its window, so it
// is looked for a few times, and it is let come forward, which Windows allows only by leave.
func Wake() error {
	for range 30 {
		hwnd, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(className)), 0)
		if hwnd != 0 {
			var pid uint32
			windows.GetWindowThreadProcessId(windows.HWND(hwnd), &pid)
			procAllowSetForegroundWindow.Call(uintptr(pid))
			procPostMessageW.Call(hwnd, wmWake, 0, 0)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return ErrInstance
}
