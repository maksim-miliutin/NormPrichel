package window

import (
	"io"
	"os"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

var (
	procMessageBoxW      = user32.NewProc("MessageBoxW")
	procCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	procAppendMenuW      = user32.NewProc("AppendMenuW")
	procTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	procDestroyMenu      = user32.NewProc("DestroyMenu")
	procChooseColorW     = comdlg32.NewProc("ChooseColorW")
	procGetOpenFileNameW = comdlg32.NewProc("GetOpenFileNameW")
)

const (
	mbYesNo       = 0x0004
	mbIconWarning = 0x0030
	mbDefButton2  = 0x0100
	idYes         = 6

	mfString = 0x0000
	mfGrayed = 0x0001

	tpmLeftAlign = 0x0000
	tpmTopAlign  = 0x0000
	tpmNoNotify  = 0x0080
	tpmReturnCmd = 0x0100

	ccRGBInit  = 0x0001
	ccFullOpen = 0x0002

	ofnHideReadOnly  = 0x00000004
	ofnNoChangeDir   = 0x00000008
	ofnPathMustExist = 0x00000800
	ofnFileMustExist = 0x00001000
	ofnExplorer      = 0x00080000
)

type chooseColor struct {
	Size         uint32
	Owner        uintptr
	Instance     uintptr
	Result       uint32
	CustomColors *[16]uint32
	Flags        uint32
	CustomData   uintptr
	Hook         uintptr
	TemplateName *uint16
}

type openFileName struct {
	Size          uint32
	Owner         uintptr
	Instance      uintptr
	Filter        *uint16
	CustomFilter  *uint16
	MaxCustFilter uint32
	FilterIndex   uint32
	File          *uint16
	MaxFile       uint32
	FileTitle     *uint16
	MaxFileTitle  uint32
	InitialDir    *uint16
	Title         *uint16
	Flags         uint32
	FileOffset    uint16
	FileExtension uint16
	DefExt        *uint16
	CustomData    uintptr
	Hook          uintptr
	TemplateName  *uint16
	Reserved      uintptr
	ReservedWord  uint32
	FlagsEx       uint32
}

func (w *Window) Text(id int) string {
	return textOf(w.children[id])
}

func (w *Window) Row(id int) int {
	row, _, _ := procSendMessageW.Call(w.children[id], lbGetCurSel, 0, 0)

	return int(int32(row))
}

func (w *Window) Confirm(question string) bool {
	words, _ := windows.UTF16PtrFromString(question)
	answer, _, _ := procMessageBoxW.Call(w.hwnd, uintptr(unsafe.Pointer(words)), uintptr(unsafe.Pointer(title)), mbYesNo|mbIconWarning|mbDefButton2)

	return answer == idYes
}

func (w *Window) Colour(start crosshair.RGB) (crosshair.RGB, bool) {
	dialog := chooseColor{Owner: w.hwnd, Result: colourRef(start), CustomColors: &w.custom, Flags: ccRGBInit | ccFullOpen}
	dialog.Size = uint32(unsafe.Sizeof(dialog))
	if chosen, _, _ := procChooseColorW.Call(uintptr(unsafe.Pointer(&dialog))); chosen == 0 {
		return start, false
	}

	return crosshair.RGB{R: uint8(dialog.Result), G: uint8(dialog.Result >> 8), B: uint8(dialog.Result >> 16)}, true
}

func (w *Window) Picture() (io.ReadCloser, error) {
	filter := utf16.Encode([]rune("PNG\x00*.png\x00\x00"))
	path := make([]uint16, windows.MAX_LONG_PATH)
	dialog := openFileName{
		Owner:   w.hwnd,
		Filter:  &filter[0],
		File:    &path[0],
		MaxFile: uint32(len(path)),
		Flags:   ofnExplorer | ofnFileMustExist | ofnPathMustExist | ofnHideReadOnly | ofnNoChangeDir,
	}
	dialog.Size = uint32(unsafe.Sizeof(dialog))
	if chosen, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&dialog))); chosen == 0 {
		return nil, nil
	}

	file, err := os.Open(windows.UTF16ToString(path))
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (w *Window) Pick(rows []string, none string) (int, bool) {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return 0, false
	}
	defer procDestroyMenu.Call(menu)

	if len(rows) == 0 {
		words, _ := windows.UTF16PtrFromString(none)
		procAppendMenuW.Call(menu, mfString|mfGrayed, 0, uintptr(unsafe.Pointer(words)))
	}
	for i, row := range rows {
		// An ampersand in a window's title would mark the next letter as the item's key.
		words, _ := windows.UTF16PtrFromString(strings.ReplaceAll(row, "&", "&&"))
		procAppendMenuW.Call(menu, mfString, uintptr(idAddMenu+i), uintptr(unsafe.Pointer(words)))
	}

	var button rect
	procGetWindowRect.Call(w.children[idAddGame], uintptr(unsafe.Pointer(&button)))
	chosen, _, _ := procTrackPopupMenu.Call(menu, tpmLeftAlign|tpmTopAlign|tpmNoNotify|tpmReturnCmd,
		uintptr(button.Left), uintptr(button.Bottom), 0, w.hwnd, 0)
	if int(chosen) < idAddMenu {
		return 0, false
	}

	return int(chosen) - idAddMenu, true
}

func (w *Window) Visit(address string) error {
	verb, _ := windows.UTF16PtrFromString("open")
	target, err := windows.UTF16PtrFromString(address)
	if err != nil {
		return err
	}

	return windows.ShellExecute(windows.Handle(w.hwnd), verb, target, nil, nil, windows.SW_SHOWNORMAL)
}
