package tray

const Background = "--background"

// StartCommand quotes the path: unquoted, a path with spaces is cut at the first one, and
// Windows would look for C:\Program instead of the program.
func StartCommand(exe string) string {
	return `"` + exe + `" ` + Background
}

// Approved reads the mark Task Manager keeps for a program that starts at sign-in: an odd first
// byte means switched off there, and no mark at all means nobody switched it off.
func Approved(mark []byte) bool {
	return len(mark) == 0 || mark[0]&1 == 0
}
