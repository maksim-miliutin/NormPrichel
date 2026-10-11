package settings

import (
	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/live"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

// The page carries no words of its own: every label comes from here, in the language of
// Windows, and a test checks that both languages name the same things.
var texts = map[string]map[string]string{
	"ru": {
		"visible":       "Прицел виден",
		"autostart":     "Запускать вместе с Windows",
		"games":         "Игры",
		"addGame":       "Добавить игру",
		"chooseWindow":  "Выберите окно игры: она должна быть запущена.",
		"noWindows":     "Подходящих окон нет. Запустите игру и нажмите «Обновить».",
		"refresh":       "Обновить",
		"cancel":        "Отмена",
		"noGames":       "Список пуст. Запустите игру в режиме «окно без рамки» и нажмите «Добавить игру».",
		"gameOff":       "выкл",
		"name":          "Название",
		"enabled":       "Включена",
		"remove":        "Удалить игру",
		"confirm":       "Удалить игру из списка? Её прицел и сдвиг пропадут.",
		"confirmForget": "Удалить этот сохранённый прицел?",

		"crosshair":        "Прицел",
		"readyMade":        "Готовый",
		"picture":          "Своя картинка",
		"shape.dot":        "Точка",
		"shape.cross":      "Крест",
		"shape.cross-dot":  "Крест с точкой",
		"shape.circle":     "Круг",
		"shape.circle-dot": "Круг с точкой",
		"shape.t":          "T",
		"color":            "Цвет",
		"opacity":          "Непрозрачность, %",
		"length":           "Длина луча, px",
		"thickness":        "Толщина, px",
		"gap":              "Зазор от центра, px",
		"radius":           "Радиус круга, px",
		"dot":              "Размер точки, px",
		"outline":          "Обводка, px (0: без обводки)",
		"outlineColor":     "Цвет обводки",
		"upload":           "Загрузить PNG",
		"uploadHint":       "PNG с прозрачностью, до 2 МБ и до 512 × 512 px. Центр картинки встанет в точку прицеливания.",
		"scale":            "Масштаб, %",

		"preview":      "Превью",
		"position":     "Положение",
		"positionHint": "Сдвиг от центра окна игры, не больше 500 px по каждой оси.",
		"step":         "Шаг, px",
		"centre":       "В центр",

		"saved":       "Сохранённые прицелы",
		"saveAs":      "Название для текущего",
		"saveCurrent": "Сохранить",
		"use":         "Выбрать",
		"forget":      "Удалить",
		"noSaved":     "Пока ничего не сохранено.",

		"hotkeys":         "Горячие клавиши",
		"hotkeysHint":     "Работают, только пока впереди окно игры из списка. Щёлкните поле и нажмите сочетание.",
		"hotkey.toggle":   "Показать или скрыть прицел, пока вы в игре",
		"hotkey.nudge":    "Сдвиг на 1 px: держите их и жмите стрелки",
		"hotkey.nudgeFar": "Сдвиг на 10 px: держите их и жмите стрелки",
		"hotkey.center":   "Вернуть в центр",
		"hotkey.next":     "Следующий сохранённый прицел",
		"taken":           "занято другой программой.",
		"take":            "Взять",

		"about":      "О программе",
		"version":    "Версия",
		"risk":       "Правила игры могут запрещать сторонний прицел. Риск бана несёт тот, кто запускает NormPrichel.",
		"borderless": "Игра должна идти в режиме «окно без рамки» или в окне: поверх эксклюзивного полного экрана прицела не видно.",

		"error.not-png":       "Это не PNG.",
		"error.too-big":       "Файл больше 2 МБ.",
		"error.too-large":     "Картинка больше 512 × 512 px.",
		"error.unknown-shape": "Такой формы прицела нет.",
		"error.no-picture":    "Картинки прицела нет в папке программы. Загрузите её снова.",
		"error.bad-hotkey":    "Сочетанию нужны Ctrl, Alt или Win и одна клавиша.",
		"error.same-hotkey":   "Это сочетание уже у другого действия.",
		"error.listed":        "Эта игра уже в списке.",
		"error.no-exe":        "У окна нет имени программы.",
		"error.no-game":       "Этой игры уже нет в списке.",
		"error.no-saved":      "Такого сохранённого прицела нет.",
		"error.refused":       "Программа не приняла изменение.",
		"error.bad-request":   "Программа не поняла страницу. Обновите её.",
		"error.failed":        "Не получилось:",
		"error.offline":       "Программа не отвечает: возможно, она закрыта.",
		"error.stale":         "Эта страница от прошлого запуска программы. Откройте настройки заново из значка в трее.",
		"error.autostart":     "Не получилось переключить запуск вместе с Windows.",
	},
	"en": {
		"visible":       "Crosshair on",
		"autostart":     "Start with Windows",
		"games":         "Games",
		"addGame":       "Add a game",
		"chooseWindow":  "Pick the game's window: the game has to be running.",
		"noWindows":     "No suitable windows. Start the game and press Refresh.",
		"refresh":       "Refresh",
		"cancel":        "Cancel",
		"noGames":       "The list is empty. Start a game in borderless window mode and press Add a game.",
		"gameOff":       "off",
		"name":          "Name",
		"enabled":       "On",
		"remove":        "Remove the game",
		"confirm":       "Remove the game from the list? Its crosshair and offset will be lost.",
		"confirmForget": "Delete this saved crosshair?",

		"crosshair":        "Crosshair",
		"readyMade":        "Ready-made",
		"picture":          "Own picture",
		"shape.dot":        "Dot",
		"shape.cross":      "Cross",
		"shape.cross-dot":  "Cross with dot",
		"shape.circle":     "Circle",
		"shape.circle-dot": "Circle with dot",
		"shape.t":          "T",
		"color":            "Colour",
		"opacity":          "Opacity, %",
		"length":           "Arm length, px",
		"thickness":        "Thickness, px",
		"gap":              "Gap from the centre, px",
		"radius":           "Circle radius, px",
		"dot":              "Dot size, px",
		"outline":          "Outline, px (0: none)",
		"outlineColor":     "Outline colour",
		"upload":           "Upload a PNG",
		"uploadHint":       "A PNG with transparency, up to 2 MB and 512 × 512 px. Its centre goes on the aim point.",
		"scale":            "Scale, %",

		"preview":      "Preview",
		"position":     "Position",
		"positionHint": "Offset from the centre of the game window, up to 500 px each way.",
		"step":         "Step, px",
		"centre":       "Centre",

		"saved":       "Saved crosshairs",
		"saveAs":      "Name for the current one",
		"saveCurrent": "Save",
		"use":         "Use",
		"forget":      "Delete",
		"noSaved":     "Nothing saved yet.",

		"hotkeys":         "Hotkeys",
		"hotkeysHint":     "They work only while a listed game is in front. Click a field and press the keys.",
		"hotkey.toggle":   "Show or hide the crosshair while you stay in the game",
		"hotkey.nudge":    "Nudge by 1 px: hold these and press arrows",
		"hotkey.nudgeFar": "Nudge by 10 px: hold these and press arrows",
		"hotkey.center":   "Back to the centre",
		"hotkey.next":     "Next saved crosshair",
		"taken":           "is held by another program.",
		"take":            "Use",

		"about":      "About",
		"version":    "Version",
		"risk":       "A game's rules may forbid a third-party crosshair. The risk of a ban lies with whoever runs NormPrichel.",
		"borderless": "The game has to run in a borderless window or a window: the crosshair cannot show over exclusive full screen.",

		"error.not-png":       "This is not a PNG.",
		"error.too-big":       "The file is over 2 MB.",
		"error.too-large":     "The picture is over 512 × 512 px.",
		"error.unknown-shape": "There is no such crosshair shape.",
		"error.no-picture":    "The crosshair picture is missing from the program's folder. Upload it again.",
		"error.bad-hotkey":    "A hotkey needs Ctrl, Alt or Win and one key.",
		"error.same-hotkey":   "Another action has this hotkey already.",
		"error.listed":        "This game is on the list already.",
		"error.no-exe":        "The window has no program name.",
		"error.no-game":       "This game is no longer on the list.",
		"error.no-saved":      "There is no such saved crosshair.",
		"error.refused":       "The program refused the change.",
		"error.bad-request":   "The program could not read the page. Reload it.",
		"error.failed":        "That did not work:",
		"error.offline":       "The program does not answer: it may have been closed.",
		"error.stale":         "This page is from an earlier run of the program. Open the settings again from the tray icon.",
		"error.autostart":     "Starting with Windows could not be switched.",
	},
}

func textFor(language string) map[string]string {
	if text, ok := texts[language]; ok {
		return text
	}

	return texts["en"]
}

// A refusal is named by its code and worded in the player's language.
var reasons = []struct {
	err  error
	code string
}{
	{crosshair.ErrNotPNG, "not-png"},
	{crosshair.ErrFileTooBig, "too-big"},
	{crosshair.ErrTooLarge, "too-large"},
	{crosshair.ErrUnknownShape, "unknown-shape"},
	{profiles.ErrNoPicture, "no-picture"},
	{hotkeys.ErrBadHotkey, "bad-hotkey"},
	{hotkeys.ErrSameHotkey, "same-hotkey"},
	{profiles.ErrListed, "listed"},
	{profiles.ErrNoExe, "no-exe"},
	{profiles.ErrNoGame, "no-game"},
	{profiles.ErrNoSaved, "no-saved"},
	{live.ErrRefused, "refused"},
	{ErrBadRequest, "bad-request"},
	{ErrAutostart, "autostart"},
}
