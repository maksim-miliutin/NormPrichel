package foreground_test

import (
	"reflect"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
)

func shown(title string, process uint32) foreground.TopWindow {
	return foreground.TopWindow{Title: title, Process: process, Visible: true}
}

func TestAPlayerChoosesFromTitledWindowsOnTheDesktop(t *testing.T) {
	hidden := shown("Hidden", 44)
	hidden.Visible = false
	tool := shown("Floating toolbar", 44)
	tool.Extended = 0x00000080
	overlay := shown("Another overlay", 44)
	overlay.Extended = 0x08000020
	cloaked := shown("Suspended store app", 44)
	cloaked.Cloaked = true
	owned := shown("Save as", 44)
	owned.Owned = true
	desktop := shown("Program Manager", 44)
	desktop.Desktop = true
	styled := shown("Styled", 44)
	styled.Extended = 0x00040100
	launched := shown("Game launched from a launcher", 44)
	launched.Owned = true
	launched.Extended = 0x00040000
	pinned := shown("Pinned tool", 44)
	pinned.Extended = 0x00040080

	top := []foreground.TopWindow{
		shown("STALZONE", 42),
		hidden, tool, overlay, cloaked, owned, desktop, styled, launched, pinned,
		shown("", 44),
		shown("   ", 44),
		shown("NormPrichel", own),
		shown("Gone", 99),
		shown("notes.txt - Notepad", 44),
	}

	got := foreground.Choices(top, processes, own)

	want := []foreground.Choice{
		{Title: "Game launched from a launcher", Exe: "notepad.exe"},
		{Title: "notes.txt - Notepad", Exe: "notepad.exe"},
		{Title: "Pinned tool", Exe: "notepad.exe"},
		{Title: "STALZONE", Exe: "STALZONE.EXE"},
		{Title: "Styled", Exe: "notepad.exe"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestChoicesComeInTitleOrderOnceEach(t *testing.T) {
	top := []foreground.TopWindow{
		shown("b", 44), shown("A", 42), shown("a", 44), shown("b", 44), shown("a", 42),
	}

	got := foreground.Choices(top, processes, own)

	want := []foreground.Choice{
		{Title: "A", Exe: "STALZONE.EXE"},
		{Title: "a", Exe: "STALZONE.EXE"},
		{Title: "a", Exe: "notepad.exe"},
		{Title: "b", Exe: "notepad.exe"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
