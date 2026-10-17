package window

import (
	"io"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/settings"
)

const (
	repository   = "https://github.com/maksim-miliutin/NormPrichel"
	pictureLimit = 4 << 20 // bytes read of a picture file; the picture rules refuse past 2 MB anyway
)

type Inputs interface {
	Text(id int) string
	Row(id int) int // -1 while no row is chosen
	Confirm(question string) bool
	Colour(start crosshair.RGB) (crosshair.RGB, bool)
	Picture() (io.ReadCloser, error) // nil when the player changed their mind
	Pick(rows []string, none string) (int, bool)
	Visit(address string) error
}

type Actions struct {
	Form   *settings.Form
	Inputs Inputs
}

func (a Actions) Click(id int) error {
	screen := a.Form.Screen()
	game := shownGame(screen)
	if shape, _, ok := rowPart(id, idShape, 1, shapeCount); ok {
		return a.Form.SetShape(crosshair.Shapes()[shape])
	}
	if swatch, _, ok := rowPart(id, idSwatch, 1, len(palette)); ok {
		return a.Form.SetColor(palette[swatch])
	}
	if swatch, _, ok := rowPart(id, idOutlineSwatch, 1, len(outlinePalette)); ok {
		return a.Form.SetOutlineColor(outlinePalette[swatch])
	}
	if row, ok := hotkeyRow(screen, id, 2); ok {
		return a.Form.TakeSuggestion(row.Field)
	}
	if language, ok := languages[id]; ok {
		return a.Form.SetLanguage(language)
	}

	switch id {
	case idVisible:
		return a.Form.SetVisible(!screen.Visible)
	case idAutostart:
		if screen.Autostart == nil {
			return nil
		}
		return a.Form.StartWithWindows(!*screen.Autostart)
	case idGitHub:
		return a.Inputs.Visit(repository)
	case idAddGame:
		return a.addGame()
	case idEnabled:
		return a.Form.Enable(!game.Enabled)
	case idRemoveGame:
		if !a.Inputs.Confirm(a.Form.Word("confirm")) {
			return nil
		}
		return a.Form.RemoveGame()
	case idResetCrosshair:
		if !a.Inputs.Confirm(a.Form.Word("confirmReset")) {
			return nil
		}
		return a.Form.ResetCrosshair()
	case idReady:
		return a.Form.ShowReady()
	case idPicture:
		a.Form.ShowPicture()
	case idColor:
		return a.colour(game.Crosshair.Color, a.Form.SetColor)
	case idOutlineColor:
		return a.colour(game.Crosshair.Outline.Color, a.Form.SetOutlineColor)
	case idUpload:
		return a.upload()
	case idLeft, idRight, idUp, idDown:
		step := pad[id]
		return a.Form.Nudge(step[0], step[1])
	case idNear, idFar:
		a.Form.SetFarStep(id == idFar)
	case idCentre:
		return a.Form.Centre()
	case idAimHide:
		return a.Form.HideWhileAiming(!game.HideWhileAiming)
	case idUseSaved:
		if row := a.Inputs.Row(idSaved); row >= 0 {
			return a.Form.UseSaved(row)
		}
	case idForgetSaved:
		if row := a.Inputs.Row(idSaved); row >= 0 && a.Inputs.Confirm(a.Form.Word("confirmForget")) {
			return a.Form.ForgetSaved(row)
		}
	case idSave:
		return a.Form.Save(a.Inputs.Text(idSaveName))
	}

	return nil
}

var pad = map[int][2]int{idLeft: {-1, 0}, idRight: {1, 0}, idUp: {0, -1}, idDown: {0, 1}}

func (a Actions) Leave(id int) error {
	screen := a.Form.Screen()
	game := shownGame(screen)
	text := a.Inputs.Text(id)

	switch id {
	case idName:
		if text == game.Name {
			return nil
		}
		return a.Form.Rename(text)
	case idX, idY:
		return a.offset(id, text, game)
	}

	row, ok := hotkeyRow(screen, id, 1)
	if !ok || text == row.Text {
		return nil
	}

	return a.Form.SetHotkey(row.Field, text)
}

func (a Actions) Enter(id int) error {
	if id == idSaveName {
		return a.Click(idSave)
	}

	return a.Leave(id)
}

func (a Actions) Select(id int) error {
	screen := a.Form.Screen()
	if id != idGames {
		return nil
	}
	if row := a.Inputs.Row(idGames); row >= 0 && row < len(screen.Games) {
		a.Form.Choose(screen.Games[row].Exe)
	}

	return nil
}

func (a Actions) Open(id int) error {
	if id != idSaved {
		return nil
	}

	return a.Click(idUseSaved)
}

func (a Actions) Slide(id, value int) error {
	game := shownGame(a.Form.Screen())
	slot, place, ok := rowPart(id, idKnob, 3, knobSlots+1)
	if !ok || place != 2 {
		return nil
	}

	switch {
	case slot == knobSlots && value != game.Scale.Value:
		return a.Form.SetScale(value)
	case slot < len(game.Knobs) && value != game.Knobs[slot].Value:
		return a.Form.SetKnob(game.Knobs[slot].Name, value)
	}

	return nil
}

func (a Actions) Press(id int, modifiers, key uint32) (bool, error) {
	row, ok := hotkeyRow(a.Form.Screen(), id, 1)
	if !ok {
		return false, nil
	}
	text, ok := hotkeys.Pressed(row.Field, modifiers, key)
	if !ok {
		return false, nil
	}

	return true, a.Form.SetHotkey(row.Field, text)
}

func (a Actions) addGame() error {
	choices, err := a.Form.WindowsToAdd()
	if err != nil {
		return err
	}

	var rows []string
	for _, choice := range choices {
		rows = append(rows, choice.Title+" — "+choice.Exe)
	}
	i, ok := a.Inputs.Pick(rows, a.Form.Word("noWindows"))
	if !ok || i < 0 || i >= len(choices) {
		return nil
	}

	return a.Form.AddGame(choices[i])
}

func (a Actions) colour(start crosshair.RGB, set func(crosshair.RGB) error) error {
	rgb, ok := a.Inputs.Colour(start)
	if !ok {
		return nil
	}

	return set(rgb)
}

func (a Actions) upload() error {
	file, err := a.Inputs.Picture()
	if err != nil || file == nil {
		return err
	}
	defer file.Close()

	picture, err := io.ReadAll(io.LimitReader(file, pictureLimit))
	if err != nil {
		return err
	}

	return a.Form.AddPicture(picture)
}

func (a Actions) offset(id int, text string, game *settings.GameView) error {
	value, ok := settings.Whole(text)
	if !ok {
		return nil
	}

	x, y := game.Offset.X, game.Offset.Y
	switch id {
	case idX:
		x = value
	case idY:
		y = value
	}
	if x == game.Offset.X && y == game.Offset.Y {
		return nil
	}

	return a.Form.SetOffset(x, y)
}

func shownGame(screen settings.Screen) *settings.GameView {
	if screen.Game == nil {
		return &settings.GameView{}
	}

	return screen.Game
}

func hotkeyRow(screen settings.Screen, id, place int) (settings.HotkeyRow, bool) {
	row, at, ok := rowPart(id, idHotkey, 4, hotkeyCount)
	if !ok || at != place || row >= len(screen.Hotkeys) {
		return settings.HotkeyRow{}, false
	}

	return screen.Hotkeys[row], true
}
