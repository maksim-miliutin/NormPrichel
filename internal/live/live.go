package live

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var (
	ErrRefused = errors.New("live: the change was refused")
	ErrClosed  = errors.New("live: the settings are closed")
)

type Saver interface {
	Save(settings profiles.Settings) error
}

// State holds the settings in use. The disk is slow next to a hotkey or the overlay thread, so
// saving happens on a goroutine of its own and never under the lock.
type State struct {
	saver  Saver
	report func(error)

	mu       sync.Mutex
	settings profiles.Settings
	last     profiles.Crosshair
	closed   bool

	dirty chan struct{}
	done  chan struct{}
}

func New(settings profiles.Settings, saver Saver, report func(error)) *State {
	s := &State{
		saver:    saver,
		report:   report,
		settings: copied(settings),
		last:     lastListed(settings),
		dirty:    make(chan struct{}, 1),
		done:     make(chan struct{}),
	}
	go s.persist()

	return s
}

func (s *State) Current() profiles.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()

	return copied(s.settings)
}

func (s *State) Last() profiles.Crosshair {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.last
}

// Change keeps what a reload from the disk would give back, so the settings in use never differ
// from the ones that survive a restart.
func (s *State) Change(edit func(profiles.Settings) (profiles.Settings, error)) (profiles.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.changed(edit)
}

// A reset forgets the crosshair chosen last as well, so a game added next starts from the default.
func (s *State) Reset() (profiles.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.changed(func(profiles.Settings) (profiles.Settings, error) { return profiles.Defaults(), nil })
	if err == nil {
		s.last = profiles.DefaultCrosshair()
	}

	return settings, err
}

func (s *State) changed(edit func(profiles.Settings) (profiles.Settings, error)) (profiles.Settings, error) {
	if s.closed {
		return copied(s.settings), ErrClosed
	}

	edited, err := edit(copied(s.settings))
	if err != nil {
		return copied(s.settings), err
	}

	normal, err := reloaded(edited)
	if err != nil {
		return copied(s.settings), fmt.Errorf("%w: %w", ErrRefused, err)
	}

	s.last = chosen(s.settings, normal, s.last)
	s.settings = normal
	select {
	case s.dirty <- struct{}{}:
	default:
	}

	return copied(normal), nil
}

func (s *State) Close() {
	s.mu.Lock()
	if !s.closed {
		s.closed = true
		close(s.dirty)
	}
	s.mu.Unlock()

	<-s.done
}

// Changes that come while a save is under way leave one mark in dirty, and the next save takes
// whatever is current by then.
func (s *State) persist() {
	defer close(s.done)

	for range s.dirty {
		s.report(s.saver.Save(s.Current()))
	}
}

func reloaded(settings profiles.Settings) (profiles.Settings, error) {
	text, err := settings.Encode()
	if err != nil {
		return profiles.Settings{}, err
	}

	return profiles.Parse(text)
}

func chosen(before, after profiles.Settings, last profiles.Crosshair) profiles.Crosshair {
	for _, game := range after.Games {
		if old, listed := before.Game(game.Exe); listed && old.Crosshair != game.Crosshair {
			return game.Crosshair
		}
	}

	return last
}

func lastListed(settings profiles.Settings) profiles.Crosshair {
	if len(settings.Games) == 0 {
		return profiles.DefaultCrosshair()
	}

	return settings.Games[len(settings.Games)-1].Crosshair
}

func copied(settings profiles.Settings) profiles.Settings {
	settings.Games = slices.Clone(settings.Games)
	settings.Saved = slices.Clone(settings.Saved)

	return settings
}
