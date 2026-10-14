package window

// Every size here is in pixels at 96 dpi, the scale Windows calls 100%.
const (
	width  = 1192
	height = 590
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

type Rect struct {
	X, Y, W, H int
}

type Place struct {
	ID    int
	Kind  Kind
	Word  string // the fixed word it shows; empty when its text comes from the settings
	Group string
	Rect  Rect
}

const (
	idVisible = 100 + iota
	idAutostart
	idVersion
	idGitHub
	idProblem
	idGamesHeading
	idGames
	idAddGame
	idSavedHeading
	idSaved
	idNoSaved
	idUseSaved
	idForgetSaved
	idSaveAs
	idSaveName
	idSave
	idNoGames
	idNameLabel
	idName
	idExe
	idEnabled
	idRemoveGame
	idCrosshairHeading
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
	idPositionHeading
	idPositionHint
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
	idAimHeading
	idAimHide
	idAimHint
	idHotkeysHeading
	idHotkeysHint
	idRisk
	idBorderless
)

// Controls that come in rows count up from a base of their own, and so do the items of a menu.
const (
	idShape         = 200
	idKnob          = 300
	idHotkey        = 400
	idSwatch        = 500
	idOutlineSwatch = 520
	idAddMenu       = 1000

	shapeCount  = 12
	knobSlots   = 6
	hotkeyCount = 5
)

const (
	col1X, col1W = 12, 220
	col2X, col2W = 244, 340
	col3X, col3W = 596, 260
	col4X, col4W = 868, 312
)

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
	// Tab visits the places in this order: the top row, then each column from the top.
	for _, part := range [][]Place{topRow(), gamesColumn(), gameHeader(), crosshairColumn(), positionColumn(), hotkeysColumn(), bottomRow()} {
		places = append(places, part...)
	}

	return places
}

func topRow() []Place {
	return []Place{
		{idVisible, Switch, "visible", always, Rect{12, 12, 150, 24}},
		{idAutostart, Switch, "autostart", autostart, Rect{170, 12, 260, 24}},
		{idVersion, Hint, "", always, Rect{968, 16, 128, 18}},
		{idGitHub, Button, "", always, Rect{1104, 12, 76, 24}},
		{idProblem, Problem, "", problem, Rect{12, 42, 1168, 22}},
	}
}

func gamesColumn() []Place {
	return []Place{
		{idGamesHeading, Heading, "games", always, Rect{col1X, 72, col1W, 18}},
		{idGames, List, "", always, Rect{col1X, 94, col1W, 172}},
		{idAddGame, Button, "addGame", always, Rect{col1X, 272, col1W, 26}},
		{idSavedHeading, Heading, "saved", game, Rect{col1X, 314, col1W, 18}},
		{idSaved, List, "", saved, Rect{col1X, 336, col1W, 102}},
		{idNoSaved, Hint, "noSaved", noSaved, Rect{col1X, 336, col1W, 34}},
		{idUseSaved, Button, "use", saved, Rect{col1X, 444, 108, 26}},
		{idForgetSaved, Button, "forget", saved, Rect{col1X + 112, 444, 108, 26}},
		{idSaveAs, Label, "saveAs", game, Rect{col1X, 480, col1W, 18}},
		{idSaveName, Edit, "", game, Rect{col1X, 500, 140, 24}},
		{idSave, Button, "saveCurrent", game, Rect{col1X + 144, 500, 76, 24}},
	}
}

func gameHeader() []Place {
	return []Place{
		{idNoGames, Hint, "noGames", noGame, Rect{col2X, 72, col2W + 12 + col3W, 40}},
		{idNameLabel, Label, "name", game, Rect{col2X, 76, 70, 18}},
		{idName, Edit, "", game, Rect{col2X + 74, 72, 220, 24}},
		{idExe, Hint, "", game, Rect{col2X + 74, 98, 220, 16}},
		{idEnabled, Switch, "enabled", game, Rect{col2X + 308, 72, 120, 24}},
		{idRemoveGame, Button, "remove", game, Rect{col3X + col3W - 126, 72, 126, 26}},
	}
}

func crosshairColumn() []Place {
	places := []Place{
		{idCrosshairHeading, Heading, "crosshair", game, Rect{col2X, 126, col2W, 18}},
		{idReady, Choice, "readyMade", game, Rect{col2X, 148, 120, 26}},
		{idPicture, Choice, "picture", game, Rect{col2X + 124, 148, 140, 26}},
	}
	for i := range shapeCount {
		places = append(places, Place{idShape + i, Choice, "", ready, Rect{col2X + i%6*57, 180 + i/6*44, 53, 40}})
	}
	places = append(places, Place{idColorLabel, Label, "color", ready, Rect{col2X, 276, 60, 18}})
	for i := range palette {
		places = append(places, Place{idSwatch + i, Swatch, "", ready, Rect{col2X + 64 + i*28, 272, 24, 24}})
	}
	places = append(places, Place{idColor, Colour, "", ready, Rect{col2X + 64 + len(palette)*28, 272, 24, 24}})
	for i := range knobSlots {
		places = append(places, knobSlot(i, 302+i*26, ready, "")...)
	}
	places = append(places, Place{idOutlineLabel, Label, "outlineColor", ready, Rect{col2X, 468, 220, 18}})
	for i := range outlinePalette {
		x := col2X + 64 + (len(palette)-len(outlinePalette)+i)*28
		places = append(places, Place{idOutlineSwatch + i, Swatch, "", ready, Rect{x, 464, 24, 24}})
	}
	places = append(places,
		Place{idOutlineColor, Colour, "", ready, Rect{col2X + 64 + len(palette)*28, 464, 24, 24}},
		Place{idUploadHint, Hint, "uploadHint", picture, Rect{col2X, 180, col2W, 50}},
		Place{idUpload, Button, "upload", picture, Rect{col2X, 236, 140, 26}},
	)

	return append(places, knobSlot(knobSlots, 268, picture, "scale")...)
}

func positionColumn() []Place {
	return []Place{
		{idPreviewHeading, Heading, "preview", game, Rect{col3X, 126, col3W, 18}},
		{idPreviewDark, Preview, "", game, Rect{col3X, 148, 126, 126}},
		{idPreviewLight, Preview, "", game, Rect{col3X + 134, 148, 126, 126}},
		{idPositionHeading, Heading, "position", game, Rect{col3X, 290, col3W, 18}},
		{idPositionHint, Hint, "positionHint", game, Rect{col3X, 310, col3W, 32}},
		{idXLabel, Label, "", game, Rect{col3X, 352, 14, 18}},
		{idX, Edit, "", game, Rect{col3X + 16, 348, 60, 24}},
		{idYLabel, Label, "", game, Rect{col3X + 88, 352, 14, 18}},
		{idY, Edit, "", game, Rect{col3X + 104, 348, 60, 24}},
		{idCentre, Button, "centre", game, Rect{col3X + 176, 348, 84, 24}},
		{idLeft, Button, "", game, Rect{col3X, 380, 26, 26}},
		{idUp, Button, "", game, Rect{col3X + 30, 380, 26, 26}},
		{idDown, Button, "", game, Rect{col3X + 60, 380, 26, 26}},
		{idRight, Button, "", game, Rect{col3X + 90, 380, 26, 26}},
		{idNear, Choice, "", game, Rect{col3X + 128, 380, 60, 26}},
		{idFar, Choice, "", game, Rect{col3X + 192, 380, 68, 26}},
		{idAimHeading, Heading, "aiming", game, Rect{col3X, 422, col3W, 18}},
		{idAimHide, Switch, "hideWhileAiming", game, Rect{col3X, 444, col3W, 24}},
		{idAimHint, Hint, "aimingHint", game, Rect{col3X, 472, col3W, 48}},
	}
}

func hotkeysColumn() []Place {
	places := []Place{
		{idHotkeysHeading, Heading, "hotkeys", always, Rect{col4X, 72, col4W, 18}},
		{idHotkeysHint, Hint, "hotkeysHint", always, Rect{col4X, 92, col4W, 48}},
	}
	for i := range hotkeyCount {
		y, id := 148+i*68, idHotkey+4*i
		places = append(places,
			Place{id, Label, "", always, Rect{col4X, y, col4W, 18}},
			Place{id + 1, Hotkey, "", always, Rect{col4X, y + 20, 150, 24}},
			Place{id + 2, Button, "", taken, Rect{col4X + 154, y + 20, col4W - 154, 24}},
			Place{id + 3, Hint, "", taken, Rect{col4X, y + 46, col4W, 16}},
		)
	}

	return places
}

func bottomRow() []Place {
	return []Place{
		{idRisk, Hint, "risk", always, Rect{12, 540, 1168, 18}},
		{idBorderless, Hint, "borderless", always, Rect{12, 560, 1168, 18}},
	}
}

// The knobs a shape is drawn from fill the slots from the top, so a shape with fewer knobs
// leaves no gaps; the picture's scale has a slot of its own under the upload button.
func knobSlot(slot, y int, group, word string) []Place {
	id := idKnob + 3*slot

	return []Place{
		{id, Label, word, group, Rect{col2X, y + 4, 150, 18}},
		{id + 1, Label, "", group, Rect{col2X + 296, y + 4, col2W - 296, 18}},
		{id + 2, Slider, "", group, Rect{col2X + 154, y, 136, 26}},
	}
}

// Edges scale rather than sizes, so controls that touch at 100% still touch at any scale.
func Scale(r Rect, dpi int) Rect {
	x, y := scaled(r.X, dpi), scaled(r.Y, dpi)

	return Rect{x, y, scaled(r.X+r.W, dpi) - x, scaled(r.Y+r.H, dpi) - y}
}

func scaled(v, dpi int) int {
	return v * dpi / 96
}
