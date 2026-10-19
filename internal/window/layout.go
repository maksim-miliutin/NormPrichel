package window

import "github.com/maksim-miliutin/NormPrichel/internal/crosshair"

// Every size here is in pixels at 96 dpi, the scale Windows calls 100%.
const (
	width  = 1200
	height = 630
)

type Kind string

const (
	Button  Kind = "button"
	Choice  Kind = "choice" // a button that stays down while its choice is made
	Switch  Kind = "switch"
	Colour  Kind = "colour" // a square that opens the colour dialog
	Swatch  Kind = "swatch" // a square of one ready colour
	Edit    Kind = "edit"
	Hotkey  Kind = "hotkey" // an edit that also takes a pressed combination
	List    Kind = "list"
	Label   Kind = "label"
	Heading Kind = "heading"
	Hint    Kind = "hint"
	Problem Kind = "problem"
	Preview Kind = "preview"
	Slider  Kind = "slider"
	Empty   Kind = "empty" // a hint in a frame, where a list has nothing to show
	Tab     Kind = "tab"   // a button that opens one part of the window
	Divider Kind = "divider"
)

const (
	always    = ""
	game      = "game"
	noGame    = "no-game"
	ready     = "ready"
	picture   = "picture"
	saved     = "saved"
	noSaved   = "no-saved"
	taken     = "taken"
	autostart = "autostart"
	problem   = "problem"
)

// The tabs a control stands on; none at all means every tab.
type Tabs uint8

const (
	crosshairTab Tabs = 1 << iota
	hotkeysTab
	generalTab
)

// The tab buttons in the order they stand; the settings keep the open tab under its word.
var tabs = []struct {
	id   int
	word string
	on   Tabs
}{
	{idTabCrosshair, "crosshair", crosshairTab},
	{idTabHotkeys, "hotkeys", hotkeysTab},
	{idTabGeneral, "general", generalTab},
}

type Rect struct {
	X, Y, W, H int
}

type Place struct {
	ID    int
	Kind  Kind
	Word  string // the fixed word it shows; empty when its text comes from the settings
	Group string
	Rect  Rect
	Tabs  Tabs
}

const (
	idVisible = 100 + iota
	idTabCrosshair
	idTabHotkeys
	idTabGeneral
	idBarLine
	idGamesLine
	idProblem
	idGamesHeading
	idGames
	idAddGame
	idNameLabel
	idName
	idExe
	idEnabled
	idRemoveGame
	idNoGames
	idReady
	idPicture
	idColorLabel
	idColor
	idOutlineLabel
	idOutlineColor
	idUploadHint
	idUpload
	idPreviewHeading
	idPreviewDark
	idPreviewLight
	idCopyCode
	idPasteCode
	idResetCrosshair
	idSavedHeading
	idSaved
	idNoSaved
	idUseSaved
	idForgetSaved
	idSaveAs
	idSaveName
	idSave
	idPositionHeading
	idXLabel
	idX
	idYLabel
	idY
	idCentre
	idLeft
	idUp
	idDown
	idRight
	idNear
	idFar
	idAimHide
	idHotkeysHint
	idAutostart
	idLanguageHeading
	idRussian
	idEnglish
	idVersion
	idGitHub
	idRisk
	idBorderless
	idResetAll
)

// Controls that come in rows count up from a base of their own, and so do the items of a menu.
const (
	idShape         = 200
	idKnob          = 300
	idHotkey        = 400
	idSwatch        = 500
	idOutlineSwatch = 520
	idAddMenu       = 1000

	knobSlots    = 6
	hotkeyCount  = 5
	shapesPerRow = 9
)

const (
	sideX, sideW   = 16, 200
	leftX, leftW   = 252, 420
	rightX, rightW = 696, 488
	top            = 96
)

var shapeCount = len(crosshair.Shapes())

func rowPart(id, base, size, rows int) (row, place int, ok bool) {
	if id < base || id >= base+size*rows {
		return 0, 0, false
	}

	return (id - base) / size, (id - base) % size, true
}

func Size() (int, int) {
	return width, height
}

func Places() []Place {
	var places []Place
	// Tab visits the places in this order: the bar at the top, the games, then each tab from the top.
	for _, part := range [][]Place{topBar(), gamesColumn(), crosshairColumn(), positionRows(), previewColumn(), hotkeysPage(), generalPage()} {
		places = append(places, part...)
	}

	return places
}

func topBar() []Place {
	places := []Place{{idVisible, Switch, "visible", always, Rect{sideX, 18, sideW, 24}, 0}}
	for i, tab := range tabs {
		places = append(places, Place{tab.id, Tab, tab.word, always, Rect{leftX + i*138, 14, 132, 36}, 0})
	}

	return append(places,
		Place{idBarLine, Divider, "", always, Rect{0, 50, width, 1}, 0},
		Place{idGamesLine, Divider, "", always, Rect{sideX + sideW + 16, 51, 1, height - 51}, 0},
		Place{idProblem, Problem, "", problem, Rect{leftX, 58, rightX + rightW - leftX, 26}, 0},
	)
}

func gamesColumn() []Place {
	return []Place{
		{idGamesHeading, Heading, "games", always, Rect{sideX, top, sideW, 18}, 0},
		{idGames, List, "", always, Rect{sideX, top + 22, sideW, 230}, 0},
		{idAddGame, Button, "addGame", always, Rect{sideX, top + 258, sideW, 28}, 0},
		{idNameLabel, Label, "name", game, Rect{sideX, top + 302, sideW, 18}, 0},
		{idName, Edit, "", game, Rect{sideX, top + 322, sideW, 28}, 0},
		{idExe, Hint, "", game, Rect{sideX, top + 352, sideW, 16}, 0},
		{idEnabled, Switch, "enabled", game, Rect{sideX, top + 376, sideW, 24}, 0},
		{idAimHide, Switch, "hideWhileAiming", game, Rect{sideX, top + 406, sideW, 24}, 0},
		{idRemoveGame, Button, "remove", game, Rect{sideX, top + 442, sideW, 28}, 0},
	}
}

func crosshairColumn() []Place {
	places := []Place{
		{idNoGames, Hint, "noGames", noGame, Rect{leftX, top, rightX + rightW - leftX, 40}, crosshairTab},
		{idReady, Choice, "readyMade", game, Rect{leftX, top, 150, 30}, crosshairTab},
		{idPicture, Choice, "picture", game, Rect{leftX + 156, top, 170, 30}, crosshairTab},
	}
	for i := range shapeCount {
		places = append(places, Place{idShape + i, Choice, "", ready, Rect{leftX + i%shapesPerRow*47, top + 42 + i/shapesPerRow*48, 44, 44}, crosshairTab})
	}
	colour := colourRow()
	places = append(places, Place{idColorLabel, Label, "color", ready, Rect{leftX, colour + 5, 64, 18}, crosshairTab})
	for i := range palette {
		places = append(places, Place{idSwatch + i, Swatch, "", ready, Rect{leftX + 68 + i*32, colour, 28, 28}, crosshairTab})
	}
	places = append(places,
		Place{idColor, Colour, "", ready, Rect{leftX + 68 + len(palette)*32, colour, 28, 28}, crosshairTab},
		Place{idOutlineLabel, Label, "outlineColor", ready, Rect{leftX, colour + 35, 220, 18}, crosshairTab},
	)
	for i := range outlinePalette {
		x := leftX + 68 + (len(palette)-len(outlinePalette)+i)*32
		places = append(places, Place{idOutlineSwatch + i, Swatch, "", ready, Rect{x, colour + 30, 28, 28}, crosshairTab})
	}
	places = append(places, Place{idOutlineColor, Colour, "", ready, Rect{leftX + 68 + len(palette)*32, colour + 30, 28, 28}, crosshairTab})
	for i := range knobSlots {
		places = append(places, knobSlot(i, colour+64+i*30, ready, "")...)
	}
	places = append(places,
		Place{idUploadHint, Hint, "uploadHint", picture, Rect{leftX, top + 42, leftW, 50}, crosshairTab},
		Place{idUpload, Button, "upload", picture, Rect{leftX, top + 98, 170, 28}, crosshairTab},
	)

	return append(places, knobSlot(knobSlots, top+134, picture, "scale")...)
}

func colourRow() int {
	return top + 42 + (shapeCount+shapesPerRow-1)/shapesPerRow*48
}

func previewColumn() []Place {
	return []Place{
		{idPreviewHeading, Heading, "preview", game, Rect{rightX, top, rightW, 18}, crosshairTab},
		{idPreviewDark, Preview, "", game, Rect{rightX, top + 22, 232, 232}, crosshairTab},
		{idPreviewLight, Preview, "", game, Rect{rightX + rightW - 232, top + 22, 232, 232}, crosshairTab},
		{idCopyCode, Button, "copyCode", game, Rect{rightX, top + 264, 156, 28}, crosshairTab},
		{idPasteCode, Button, "pasteCode", game, Rect{rightX + 162, top + 264, 156, 28}, crosshairTab},
		{idResetCrosshair, Button, "resetCrosshair", game, Rect{rightX + 324, top + 264, rightW - 324, 28}, crosshairTab},
		{idSavedHeading, Heading, "saved", game, Rect{rightX, top + 306, rightW, 18}, crosshairTab},
		{idSaved, List, "", saved, Rect{rightX, top + 328, rightW, 84}, crosshairTab},
		{idNoSaved, Empty, "noSaved", noSaved, Rect{rightX, top + 328, rightW, 118}, crosshairTab},
		{idUseSaved, Button, "use", saved, Rect{rightX, top + 418, 240, 28}, crosshairTab},
		{idForgetSaved, Button, "forget", saved, Rect{rightX + 248, top + 418, rightW - 248, 28}, crosshairTab},
		{idSaveAs, Label, "saveAs", game, Rect{rightX, top + 456, rightW, 18}, crosshairTab},
		{idSaveName, Edit, "", game, Rect{rightX, top + 476, 376, 28}, crosshairTab},
		{idSave, Button, "saveCurrent", game, Rect{rightX + 382, top + 476, rightW - 382, 28}, crosshairTab},
	}
}

// The offset belongs to the game, not to the look, so it stays under the knobs for a picture too.
func positionRows() []Place {
	y := colourRow() + 64 + knobSlots*30 + 12

	return []Place{
		{idPositionHeading, Heading, "position", game, Rect{leftX, y, leftW, 18}, crosshairTab},
		{idXLabel, Label, "", game, Rect{leftX, y + 27, 14, 18}, crosshairTab},
		{idX, Edit, "", game, Rect{leftX + 18, y + 22, 76, 28}, crosshairTab},
		{idYLabel, Label, "", game, Rect{leftX + 108, y + 27, 14, 18}, crosshairTab},
		{idY, Edit, "", game, Rect{leftX + 126, y + 22, 76, 28}, crosshairTab},
		{idCentre, Button, "centre", game, Rect{leftX + 216, y + 22, 120, 28}, crosshairTab},
		{idLeft, Button, "", game, Rect{leftX, y + 58, 34, 30}, crosshairTab},
		{idUp, Button, "", game, Rect{leftX + 40, y + 58, 34, 30}, crosshairTab},
		{idDown, Button, "", game, Rect{leftX + 80, y + 58, 34, 30}, crosshairTab},
		{idRight, Button, "", game, Rect{leftX + 120, y + 58, 34, 30}, crosshairTab},
		{idNear, Choice, "", game, Rect{leftX + 170, y + 58, 76, 30}, crosshairTab},
		{idFar, Choice, "", game, Rect{leftX + 252, y + 58, 84, 30}, crosshairTab},
	}
}

func hotkeysPage() []Place {
	places := []Place{{idHotkeysHint, Hint, "hotkeysHint", always, Rect{leftX, top, 600, 34}, hotkeysTab}}
	for i := range hotkeyCount {
		y, id := top+48+i*80, idHotkey+4*i
		places = append(places,
			Place{id, Label, "", always, Rect{leftX, y, 600, 18}, hotkeysTab},
			Place{id + 1, Hotkey, "", always, Rect{leftX, y + 22, 220, 28}, hotkeysTab},
			Place{id + 2, Button, "", taken, Rect{leftX + 228, y + 22, 260, 28}, hotkeysTab},
			Place{id + 3, Hint, "", taken, Rect{leftX, y + 52, 600, 16}, hotkeysTab},
		)
	}

	return places
}

func generalPage() []Place {
	return []Place{
		{idAutostart, Switch, "autostart", autostart, Rect{leftX, top, leftW, 24}, generalTab},
		{idLanguageHeading, Heading, "language", always, Rect{leftX, top + 44, leftW, 18}, generalTab},
		{idRussian, Choice, "", always, Rect{leftX, top + 66, 130, 28}, generalTab},
		{idEnglish, Choice, "", always, Rect{leftX + 136, top + 66, 130, 28}, generalTab},
		{idGitHub, Button, "", always, Rect{leftX, top + 112, 110, 28}, generalTab},
		{idVersion, Hint, "", always, Rect{leftX + 122, top + 117, 200, 18}, generalTab},
		{idRisk, Hint, "risk", always, Rect{leftX, top + 164, rightX + rightW - leftX, 18}, generalTab},
		{idBorderless, Hint, "borderless", always, Rect{leftX, top + 186, rightX + rightW - leftX, 18}, generalTab},
		{idResetAll, Button, "resetAll", always, Rect{leftX, top + 226, 240, 28}, generalTab},
	}
}

// The knobs a shape is drawn from fill the slots from the top, so a shape with fewer knobs
// leaves no gaps; the picture's scale has a slot of its own under the upload button.
func knobSlot(slot, y int, group, word string) []Place {
	id := idKnob + 3*slot

	return []Place{
		{id, Label, word, group, Rect{leftX, y + 6, 150, 18}, crosshairTab},
		{id + 1, Label, "", group, Rect{leftX + 370, y + 6, leftW - 370, 18}, crosshairTab},
		{id + 2, Slider, "", group, Rect{leftX + 154, y, 210, 30}, crosshairTab},
	}
}

// Fields, lists and the hint of an empty list are drawn borderless inside a frame the window
// paints around them; a field keeps sixteen pixels of height, a line of text.
func Interior(p Place) Rect {
	r := p.Rect
	switch p.Kind {
	case Edit, Hotkey:
		return Rect{r.X + 7, r.Y + (r.H-16)/2, r.W - 14, 16}
	case List:
		return Rect{r.X + 2, r.Y + 2, r.W - 4, r.H - 4}
	case Empty:
		return Rect{r.X + 8, r.Y + 6, r.W - 16, r.H - 12}
	}

	return r
}

// Edges scale rather than sizes, so controls that touch at 100% still touch at any scale.
func Scale(r Rect, dpi int) Rect {
	x, y := scaled(r.X, dpi), scaled(r.Y, dpi)

	return Rect{x, y, scaled(r.X+r.W, dpi) - x, scaled(r.Y+r.H, dpi) - y}
}

func scaled(v, dpi int) int {
	return v * dpi / 96
}
