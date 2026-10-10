package profiles

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	ErrNoGame  = errors.New("profiles: the game is not on the list")
	ErrListed  = errors.New("profiles: the game is on the list already")
	ErrNoExe   = errors.New("profiles: a game is known by its exe, and this one has none")
	ErrNoSaved = errors.New("profiles: no saved crosshair under that number")
)

func (s Settings) WithGame(game Game) (Settings, error) {
	if game.Exe == "" {
		return s, ErrNoExe
	}
	if s.index(game.Exe) >= 0 {
		return s, fmt.Errorf("%w: %q", ErrListed, game.Exe)
	}

	s.Games = append(slices.Clone(s.Games), game)

	return s, nil
}

func (s Settings) WithoutGame(exe string) (Settings, error) {
	i := s.index(exe)
	if i < 0 {
		return s, fmt.Errorf("%w: %q", ErrNoGame, exe)
	}

	s.Games = slices.Delete(slices.Clone(s.Games), i, i+1)

	return s, nil
}

func (s Settings) Edited(exe string, edit func(Game) Game) (Settings, error) {
	i := s.index(exe)
	if i < 0 {
		return s, fmt.Errorf("%w: %q", ErrNoGame, exe)
	}

	games := slices.Clone(s.Games)
	games[i] = edit(games[i])
	games[i].Offset = games[i].Offset.clamped()
	s.Games = games

	return s, nil
}

func (s Settings) WithSaved(entry Saved) Settings {
	s.Saved = append(slices.Clone(s.Saved), entry)

	return s
}

func (s Settings) WithoutSaved(index int) (Settings, error) {
	if index < 0 || index >= len(s.Saved) {
		return s, fmt.Errorf("%w: %d", ErrNoSaved, index)
	}

	s.Saved = slices.Delete(slices.Clone(s.Saved), index, index+1)

	return s, nil
}

// NextSaved steps over saved copies of the current crosshair: the same picture twice in a row
// would look like a key that did nothing.
func (s Settings) NextSaved(current Crosshair) (Crosshair, bool) {
	start := slices.IndexFunc(s.Saved, func(entry Saved) bool { return entry.Crosshair == current })
	for step := 1; step <= len(s.Saved); step++ {
		if next := s.Saved[(start+step)%len(s.Saved)].Crosshair; next != current {
			return next, true
		}
	}

	return current, false
}

func (s Settings) index(exe string) int {
	return slices.IndexFunc(s.Games, func(game Game) bool { return strings.EqualFold(game.Exe, exe) })
}
