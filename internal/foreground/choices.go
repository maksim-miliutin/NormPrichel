package foreground

import (
	"cmp"
	"slices"
	"strings"
)

type TopWindow struct {
	Title    string
	Process  uint32
	Visible  bool
	Extended uint32 // the WS_EX_ style bits
	Cloaked  bool   // hidden by the shell, like a suspended store app
	Owned    bool   // a dialog of a window on the screen
	Desktop  bool   // the desktop itself, which Windows lists among the windows
}

// Alt+Tab leaves out tool windows and dialogs unless they ask to be in, and a window that never
// activates can never be in front.
const (
	toolWindow = 0x00000080
	appWindow  = 0x00040000
	noActivate = 0x08000000
)

type Choice struct {
	Title string `json:"title"`
	Exe   string `json:"exe"`
}

// Choices are the windows a player could mean by "the game": the ones Alt+Tab would show, less
// this program's own.
func Choices(top []TopWindow, processes []Process, own uint32) []Choice {
	exes := make(map[uint32]string, len(processes))
	for _, p := range processes {
		exes[p.ID] = p.Exe
	}

	choices := []Choice{}
	for _, w := range top {
		exe, alive := exes[w.Process]
		if !alive || !w.offered(own) {
			continue
		}
		choices = append(choices, Choice{Title: strings.TrimSpace(w.Title), Exe: exe})
	}

	slices.SortFunc(choices, func(a, b Choice) int {
		return cmp.Or(
			strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title)),
			strings.Compare(a.Title, b.Title),
			strings.Compare(a.Exe, b.Exe),
		)
	})

	return slices.Compact(choices)
}

func (w TopWindow) offered(own uint32) bool {
	listed := w.Extended&appWindow != 0 || w.Extended&toolWindow == 0 && !w.Owned
	switchable := w.Visible && listed && w.Extended&noActivate == 0 && !w.Cloaked && !w.Desktop

	return switchable && w.Process != own && strings.TrimSpace(w.Title) != ""
}
