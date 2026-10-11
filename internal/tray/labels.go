package tray

import (
	"image"
	"image/draw"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

type Labels struct {
	Settings  string
	Toggle    string
	Autostart string
	Exit      string
}

var labels = map[string]Labels{
	"ru": {Settings: "Настройки", Toggle: "Показать или скрыть прицел", Autostart: "Запускать вместе с Windows", Exit: "Выход"},
	"en": {Settings: "Settings", Toggle: "Show or hide the crosshair", Autostart: "Start with Windows", Exit: "Exit"},
}

const langRussian = 0x19

// Language reads a Windows language identifier, whose low ten bits name the language and the
// rest its country.
func Language(id uint16) string {
	if id&0x3FF == langRussian {
		return "ru"
	}

	return "en"
}

// StartCommand quotes the path: unquoted, a path with spaces is cut at the first one, and
// Windows would look for C:\Program instead of the program.
func StartCommand(exe string) string {
	return `"` + exe + `"`
}

func LabelsFor(language string) Labels {
	if l, ok := labels[language]; ok {
		return l
	}

	return labels["en"]
}

// Icon draws the default crosshair to fit a square of the given side, as blue, green, red and
// alpha rows from the top; an icon takes its alpha straight, not multiplied in.
func Icon(side int) ([]byte, error) {
	style := crosshair.Default()
	style.Thickness = max(2, side/8)
	style.Gap = side / 8
	style.Length = side / 4
	style.DotSize = style.Thickness
	style.Outline.Width = max(1, side/16)

	drawn, err := crosshair.Draw(style)
	if err != nil {
		return nil, err
	}

	canvas := image.NewNRGBA(image.Rect(0, 0, side, side))
	corner := image.Pt(side-drawn.Rect.Dx(), side-drawn.Rect.Dy()).Div(2)
	draw.Draw(canvas, drawn.Rect.Add(corner), drawn, image.Point{}, draw.Src)

	pixels := make([]byte, len(canvas.Pix))
	for i := 0; i < len(pixels); i += 4 {
		pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = canvas.Pix[i+2], canvas.Pix[i+1], canvas.Pix[i], canvas.Pix[i+3]
	}

	return pixels, nil
}
