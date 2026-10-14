package mouse

import "golang.org/x/sys/windows"

var procGetAsyncKeyState = windows.NewLazySystemDLL("user32.dll").NewProc("GetAsyncKeyState")

const vkRButton = 0x02

type System struct{}

// The physical right button, not the one Windows swaps in for the left hand: games that read
// raw input see the mouse as it lies under the fingers too.
func (System) Aiming() bool {
	state, _, _ := procGetAsyncKeyState.Call(vkRButton)

	return held(state)
}
