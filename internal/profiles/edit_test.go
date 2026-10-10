package profiles_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

func crosshairOf(shape crosshair.Shape) profiles.Crosshair {
	style := specStyle
	style.Shape = shape

	return profiles.Crosshair{Style: style, Scale: 100}
}

var (
	dotted  = crosshairOf(crosshair.Dot)
	crossed = crosshairOf(crosshair.Cross)
	ringed  = crosshairOf(crosshair.Circle)
)

func twoGames() profiles.Settings {
	settings := specDefaults
	settings.Games = []profiles.Game{
		{Exe: "stalzone.exe", Name: "STALZONE", Enabled: true, Crosshair: dotted},
		{Exe: "other.exe", Name: "Other", Enabled: true, Crosshair: crossed, Offset: profiles.Offset{X: 4, Y: -2}},
	}
	settings.Saved = []profiles.Saved{{Name: "dot", Crosshair: dotted}, {Name: "cross", Crosshair: crossed}}

	return settings
}

func renamed(name string) func(profiles.Game) profiles.Game {
	return func(g profiles.Game) profiles.Game {
		g.Name = name
		return g
	}
}

func TestAGameJoinsTheEndOfTheList(t *testing.T) {
	added := profiles.Game{Exe: "new.exe", Name: "New", Enabled: true, Crosshair: ringed}

	got, err := twoGames().WithGame(added)
	if err != nil {
		t.Fatal(err)
	}

	if want := append(twoGames().Games, added); !reflect.DeepEqual(got.Games, want) {
		t.Errorf("games %+v, want %+v", got.Games, want)
	}
}

func TestAGameIsListedOnceWhateverTheCaseOfItsExe(t *testing.T) {
	_, err := twoGames().WithGame(profiles.Game{Exe: "STALZONE.EXE"})
	if !errors.Is(err, profiles.ErrListed) {
		t.Errorf("got %v, want ErrListed", err)
	}
}

func TestAGameWithoutAnExeIsRefused(t *testing.T) {
	_, err := twoGames().WithGame(profiles.Game{Name: "nameless"})
	if !errors.Is(err, profiles.ErrNoExe) {
		t.Errorf("got %v, want ErrNoExe", err)
	}
}

func TestAGameLeavesTheListByItsExeInAnyCase(t *testing.T) {
	got, err := twoGames().WithoutGame("STALZONE.exe")
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Games) != 1 || got.Games[0].Exe != "other.exe" {
		t.Errorf("games %+v, want only other.exe", got.Games)
	}
}

func TestAnEditReachesItsGameAlone(t *testing.T) {
	got, err := twoGames().Edited("Other.EXE", renamed("Renamed"))
	if err != nil {
		t.Fatal(err)
	}

	want := twoGames().Games
	want[1].Name = "Renamed"
	if !reflect.DeepEqual(got.Games, want) {
		t.Errorf("games %+v, want %+v", got.Games, want)
	}
}

func TestAnEditCannotTakeTheCrosshairPast500PixelsFromTheCentre(t *testing.T) {
	got, err := twoGames().Edited("other.exe", func(g profiles.Game) profiles.Game {
		g.Offset = profiles.Offset{X: g.Offset.X - 1000, Y: g.Offset.Y + 700}
		return g
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := (profiles.Offset{X: -500, Y: 500}); got.Games[1].Offset != want {
		t.Errorf("offset %+v, want %+v", got.Games[1].Offset, want)
	}
}

func TestAGameOffTheListCannotBeEditedOrRemoved(t *testing.T) {
	if _, err := twoGames().Edited("notepad.exe", renamed("x")); !errors.Is(err, profiles.ErrNoGame) {
		t.Errorf("edit: got %v, want ErrNoGame", err)
	}
	if _, err := twoGames().WithoutGame("notepad.exe"); !errors.Is(err, profiles.ErrNoGame) {
		t.Errorf("removal: got %v, want ErrNoGame", err)
	}
}

func TestACrosshairIsSavedAtTheEndAndRemovedByItsNumber(t *testing.T) {
	saved := twoGames().WithSaved(profiles.Saved{Name: "ring", Crosshair: ringed})
	if names := savedNames(saved); !slices.Equal(names, []string{"dot", "cross", "ring"}) {
		t.Fatalf("saved %v, want dot, cross, ring", names)
	}

	removed, err := saved.WithoutSaved(1)
	if err != nil {
		t.Fatal(err)
	}
	if names := savedNames(removed); !slices.Equal(names, []string{"dot", "ring"}) {
		t.Errorf("saved %v after removing the second, want dot, ring", names)
	}
}

func TestOnlyANumberOnTheSavedListCanBeRemoved(t *testing.T) {
	for _, index := range []int{-1, 2} {
		if _, err := twoGames().WithoutSaved(index); !errors.Is(err, profiles.ErrNoSaved) {
			t.Errorf("number %d: got %v, want ErrNoSaved", index, err)
		}
	}
}

// The bug this guards: an append or a delete that works on the slice it was handed writes into
// the array behind it, and one settings value changes when another is made from it.
func TestChangesLeaveTheSettingsTheyStartFrom(t *testing.T) {
	base := twoGames()
	base.Games = slices.Grow(base.Games, 4)
	base.Saved = slices.Grow(base.Saved, 4)
	before := twoGames()

	first, _ := base.WithGame(profiles.Game{Exe: "first.exe"})
	second, _ := base.WithGame(profiles.Game{Exe: "second.exe"})
	if first.Games[2].Exe != "first.exe" || second.Games[2].Exe != "second.exe" {
		t.Errorf("two games added to one list came out as %q and %q", first.Games[2].Exe, second.Games[2].Exe)
	}

	one := base.WithSaved(profiles.Saved{Name: "one"})
	two := base.WithSaved(profiles.Saved{Name: "two"})
	if one.Saved[2].Name != "one" || two.Saved[2].Name != "two" {
		t.Errorf("two crosshairs saved to one list came out as %q and %q", one.Saved[2].Name, two.Saved[2].Name)
	}

	base.Edited("stalzone.exe", renamed("changed"))
	base.WithoutGame("stalzone.exe")
	base.WithoutSaved(0)
	if !reflect.DeepEqual(base, before) {
		t.Errorf("the settings changed under the edits:\n%+v\nwant\n%+v", base, before)
	}
}

func TestTheNextSavedCrosshairComesRoundTheList(t *testing.T) {
	cases := []struct {
		name    string
		saved   []profiles.Crosshair
		current profiles.Crosshair
		want    profiles.Crosshair
		found   bool
	}{
		{name: "from the first", saved: []profiles.Crosshair{dotted, crossed, ringed}, current: dotted, want: crossed, found: true},
		{name: "from the last", saved: []profiles.Crosshair{dotted, crossed, ringed}, current: ringed, want: dotted, found: true},
		{name: "from one not saved", saved: []profiles.Crosshair{crossed, ringed}, current: dotted, want: crossed, found: true},
		{name: "to the only one saved", saved: []profiles.Crosshair{crossed}, current: dotted, want: crossed, found: true},
		{name: "past a copy of the current", saved: []profiles.Crosshair{dotted, dotted, crossed}, current: dotted, want: crossed, found: true},
		{name: "back over a copy", saved: []profiles.Crosshair{dotted, crossed, dotted}, current: crossed, want: dotted, found: true},
		{name: "with nothing else saved", saved: []profiles.Crosshair{dotted, dotted}, current: dotted, want: dotted, found: false},
		{name: "with nothing saved", saved: nil, current: dotted, want: dotted, found: false},
	}

	for _, c := range cases {
		settings := specDefaults
		for _, saved := range c.saved {
			settings = settings.WithSaved(profiles.Saved{Crosshair: saved})
		}

		got, found := settings.NextSaved(c.current)
		if got != c.want || found != c.found {
			t.Errorf("%s: got %v, %v; want %v, %v", c.name, got.Shape, found, c.want.Shape, c.found)
		}
	}
}

func savedNames(settings profiles.Settings) []string {
	names := make([]string, len(settings.Saved))
	for i, entry := range settings.Saved {
		names[i] = entry.Name
	}

	return names
}
