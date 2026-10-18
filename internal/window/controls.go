package window

import (
	"slices"
	"strconv"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/settings"
)

// The colours a swatch gives in one click; the first is the crosshair's default green.
var (
	palette = []crosshair.RGB{
		{R: 0x00, G: 0xFF, B: 0x66},
		{R: 0xFF, G: 0x00, B: 0x00},
		{R: 0xFF, G: 0xFF, B: 0x00},
		{R: 0x00, G: 0xFF, B: 0xFF},
		{R: 0xFF, G: 0x00, B: 0xFF},
		{R: 0xFF, G: 0x80, B: 0x00},
		{R: 0xFF, G: 0xFF, B: 0xFF},
		{R: 0x00, G: 0x00, B: 0x00},
	}
	outlinePalette = []crosshair.RGB{{R: 0x00, G: 0x00, B: 0x00}, {R: 0xFF, G: 0xFF, B: 0xFF}}
)

// Each language is named in itself, so a player lost in the other one still finds their own.
var languages = map[int]string{idRussian: "ru", idEnglish: "en"}

type Control struct {
	Place
	Text   string
	Shown  bool
	On     bool // a switch that is on, a choice that is made, a hotkey another program holds
	Rows   []string
	Row    int // the chosen row of a list; -1 leaves the choice to the player
	Colour crosshair.RGB
	Shape  crosshair.Shape // the shape a button draws instead of words
	Knob   settings.KnobView
	Typed  bool // the player's own words, which no change of the settings overwrites
}

func Controls(screen settings.Screen, word func(string) string) []Control {
	view := screen.Game
	shown := map[string]bool{
		always:    true,
		game:      view != nil,
		noGame:    view == nil,
		ready:     view != nil && !view.Picture,
		picture:   view != nil && view.Picture,
		saved:     view != nil && len(screen.Saved) > 0,
		noSaved:   view != nil && len(screen.Saved) == 0,
		autostart: screen.Autostart != nil,
		taken:     true,
		problem:   screen.Problem != "",
	}
	if view == nil {
		view = &settings.GameView{}
	}
	open := openTab(screen.Tab)

	var controls []Control
	for _, p := range Places() {
		c := Control{Place: p, Shown: shown[p.Group] && (p.Tabs == 0 || p.Tabs&open != 0), Row: -1}
		if p.Word != "" {
			c.Text = word(p.Word)
		}
		hotkey, hotkeyPlace, isHotkey := rowPart(p.ID, idHotkey, 4, hotkeyCount)
		slot, slotPlace, isKnob := rowPart(p.ID, idKnob, 3, knobSlots+1)
		shapeIndex, _, isShape := rowPart(p.ID, idShape, 1, shapeCount)
		swatch, _, isSwatch := rowPart(p.ID, idSwatch, 1, len(palette))
		edging, _, isEdging := rowPart(p.ID, idOutlineSwatch, 1, len(outlinePalette))
		switch {
		case isSwatch:
			c = swatchPart(c, palette[swatch], view.Crosshair.Color)
		case isEdging:
			c = swatchPart(c, outlinePalette[edging], view.Crosshair.Outline.Color)
		case isHotkey:
			c = hotkeyPart(c, hotkey, hotkeyPlace, screen.Hotkeys, word)
		case isKnob:
			c = knobPart(c, slot, slotPlace, view)
		case isShape:
			shape := crosshair.Shapes()[shapeIndex]
			c.Text, c.Shape, c.On = word("shape."+string(shape)), shape, view.Crosshair.Shape == shape
		default:
			c = filled(c, screen, view, word)
		}
		controls = append(controls, c)
	}

	return controls
}

func filled(c Control, screen settings.Screen, view *settings.GameView, word func(string) string) Control {
	switch c.ID {
	case idVisible:
		c.On = screen.Visible
	case idAutostart:
		c.On = screen.Autostart != nil && *screen.Autostart
	case idVersion:
		c.Text = word("version") + " " + screen.Version
	case idGitHub:
		c.Text = "GitHub"
	case idProblem:
		c.Text = screen.Problem
	case idGames:
		for i, row := range screen.Games {
			c.Rows = append(c.Rows, gameRow(row, word))
			if row.Chosen {
				c.Row = i
			}
		}
	case idSaved:
		c.Rows = screen.Saved
	case idSaveName:
		c.Typed = true
	case idName:
		c.Text = view.Name
	case idExe:
		c.Text = view.Exe
	case idEnabled:
		c.On = view.Enabled
	case idReady:
		c.On = !view.Picture
	case idPicture:
		c.On = view.Picture
	case idColor:
		c.Colour, c.On = view.Crosshair.Color, !slices.Contains(palette, view.Crosshair.Color)
	case idOutlineColor:
		c.Colour, c.On = view.Crosshair.Outline.Color, !slices.Contains(outlinePalette, view.Crosshair.Outline.Color)
	case idX:
		c.Text = strconv.Itoa(view.Offset.X)
	case idY:
		c.Text = strconv.Itoa(view.Offset.Y)
	case idNear:
		c.On = !screen.FarStep
	case idFar:
		c.On = screen.FarStep
	case idAimHide:
		c.On = view.HideWhileAiming
	case idCopyCode:
		c.On = screen.Copied
		if c.On {
			c.Text = word("copied")
		}
	}
	if language, ok := languages[c.ID]; ok {
		c.On = screen.Language == language
	}
	for _, tab := range tabs {
		if tab.id == c.ID {
			c.On = tab.on == openTab(screen.Tab)
		}
	}
	if text, ok := signs[c.ID]; ok {
		c.Text = text
	}

	return c
}

// A tab the window does not know, the empty one of a first start among them, opens the first.
func openTab(word string) Tabs {
	for _, tab := range tabs {
		if tab.word == word {
			return tab.on
		}
	}

	return tabs[0].on
}

func swatchPart(c Control, colour, current crosshair.RGB) Control {
	name, _ := colour.MarshalText()
	c.Text, c.Colour, c.On = string(name), colour, colour == current

	return c
}

var signs = map[int]string{
	idRussian:      "Русский",
	idEnglish:      "English",
	idColor:        "…",
	idOutlineColor: "…",
	idXLabel:       "X",
	idYLabel:       "Y",
	idLeft:         "←",
	idUp:           "↑",
	idDown:         "↓",
	idRight:        "→",
	idNear:         "1 px",
	idFar:          "10 px",
}

func gameRow(row settings.GameRow, word func(string) string) string {
	if row.Enabled {
		return row.Name
	}

	return row.Name + " · " + word("gameOff")
}

func knobPart(c Control, slot, part int, view *settings.GameView) Control {
	knob := view.Scale
	if slot < knobSlots {
		c.Shown = c.Shown && slot < len(view.Knobs)
		if !c.Shown {
			return c
		}
		knob = view.Knobs[slot]
	}

	switch part {
	case 0:
		c.Text = knob.Label
	case 1:
		c.Text = strconv.Itoa(knob.Value)
	case 2:
		c.Knob = knob
	}

	return c
}

func hotkeyPart(c Control, i, part int, rows []settings.HotkeyRow, word func(string) string) Control {
	if i >= len(rows) {
		c.Shown = false
		return c
	}

	row := rows[i]
	switch part {
	case 0:
		c.Text = row.Label
	case 1:
		c.Text, c.On = row.Text, row.Taken != ""
	case 2:
		c.Text, c.Shown = word("take")+" "+row.Suggestion, c.Shown && row.Taken != "" && row.Suggestion != ""
	case 3:
		c.Text, c.Shown = row.Taken+" "+word("taken"), c.Shown && row.Taken != ""
	}

	return c
}
