package mouse

// GetAsyncKeyState answers with a SHORT whose top bit is the button held now; its low bit is a
// press since some earlier look, which would flash the crosshair off after every click.
func held(state uintptr) bool {
	return state&0x8000 != 0
}
