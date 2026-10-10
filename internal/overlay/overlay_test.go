package overlay_test

import (
	"errors"
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/overlay"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var errBroken = errors.New("broken")

// screen records what the controller asks of the overlay window.
type screen struct {
	calls []string
	shown overlay.Bitmap
	at    image.Point
	fail  error
}

func (s *screen) Show(bitmap overlay.Bitmap, at image.Point) error {
	s.calls = append(s.calls, "show")
	if s.fail != nil {
		return s.fail
	}
	s.shown, s.at = bitmap, at

	return nil
}

func (s *screen) Move(at image.Point) error {
	s.calls = append(s.calls, "move")
	s.at = at

	return nil
}

func (s *screen) Hide() error {
	s.calls = append(s.calls, "hide")

	return nil
}

// scene hands out one target, or none, or an error.
type scene struct {
	target foreground.Target
	there  bool
	fail   error
}

func (s *scene) Target(profiles.Settings) (foreground.Target, bool, error) {
	return s.target, s.there, s.fail
}

// painter draws a blank square of the given side and counts how often it is asked to.
type painter struct {
	side  int
	calls int
	fail  error
}

func (p *painter) paint(profiles.Crosshair) (*image.RGBA, error) {
	p.calls++
	if p.fail != nil {
		return nil, p.fail
	}

	return image.NewRGBA(image.Rect(0, 0, p.side, p.side)), nil
}

func game(offset profiles.Offset) profiles.Game {
	return profiles.Game{Exe: "stalzone.exe", Enabled: true, Crosshair: profiles.Crosshair{Style: crosshair.Default(), Scale: 100}, Offset: offset}
}

var hd = image.Rect(100, 200, 1380, 920)

func TestTheCrosshairLandsOnTheCentreOfTheClientArea(t *testing.T) {
	cases := []struct {
		name   string
		client image.Rectangle
		side   int
		offset profiles.Offset
		want   image.Point
	}{
		{name: "even side: its middle corner on the middle corner", client: hd, side: 24, want: image.Pt(728, 548)},
		{name: "odd side: its middle pixel on the pixel right and below the middle", client: hd, side: 21, want: image.Pt(730, 550)},
		{name: "the offset shifts it", client: hd, side: 24, offset: profiles.Offset{X: 5, Y: -7}, want: image.Pt(733, 541)},
		{name: "an odd client area rounds its middle up and left", client: image.Rect(0, 0, 1281, 721), side: 24, want: image.Pt(628, 348)},
	}

	for _, c := range cases {
		s := &screen{}
		p := &painter{side: c.side}
		controller := overlay.NewController(s, &scene{target: foreground.Target{Game: game(c.offset), Client: c.client}, there: true}, p.paint)

		if err := controller.Refresh(profiles.Defaults()); err != nil {
			t.Fatal(err)
		}
		if s.at != c.want || s.shown.Pixels == nil {
			t.Errorf("%s: shown %v at %v, want at %v", c.name, s.shown.Pixels != nil, s.at, c.want)
		}
	}
}

func TestPixelsGoOutAsPremultipliedBlueGreenRedAlphaRowByRow(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.SetRGBA(0, 0, color.RGBA{R: 1, G: 2, B: 3, A: 4})
	img.SetRGBA(2, 0, color.RGBA{R: 5, G: 6, B: 7, A: 8})
	img.SetRGBA(1, 1, color.RGBA{R: 9, G: 10, B: 11, A: 12})
	framed := img.SubImage(image.Rect(1, 0, 3, 2)).(*image.RGBA)

	s := &screen{}
	paint := func(profiles.Crosshair) (*image.RGBA, error) { return framed, nil }
	controller := overlay.NewController(s, &scene{target: foreground.Target{Game: game(profiles.Offset{}), Client: hd}, there: true}, paint)
	if err := controller.Refresh(profiles.Defaults()); err != nil {
		t.Fatal(err)
	}

	want := overlay.Bitmap{
		Pixels: []byte{
			0, 0, 0, 0, 7, 6, 5, 8,
			11, 10, 9, 12, 0, 0, 0, 0,
		},
		Size: image.Pt(2, 2),
	}
	if !slices.Equal(s.shown.Pixels, want.Pixels) || s.shown.Size != want.Size {
		t.Errorf("got %v over %v, want %v over %v", s.shown.Pixels, s.shown.Size, want.Pixels, want.Size)
	}
}

func TestAStillSceneTouchesNothing(t *testing.T) {
	s := &screen{}
	p := &painter{side: 24}
	controller := overlay.NewController(s, &scene{target: foreground.Target{Game: game(profiles.Offset{}), Client: hd}, there: true}, p.paint)

	for i := 0; i < 5; i++ {
		if err := controller.Refresh(profiles.Defaults()); err != nil {
			t.Fatal(err)
		}
	}

	if len(s.calls) != 1 || p.calls != 1 {
		t.Errorf("calls %v and %d paintings, want one show and one painting", s.calls, p.calls)
	}
}

func TestAMovedWindowMovesTheCrosshairWithoutRedrawing(t *testing.T) {
	s := &screen{}
	p := &painter{side: 24}
	sc := &scene{target: foreground.Target{Game: game(profiles.Offset{}), Client: hd}, there: true}
	controller := overlay.NewController(s, sc, p.paint)
	if err := controller.Refresh(profiles.Defaults()); err != nil {
		t.Fatal(err)
	}

	sc.target.Client = hd.Add(image.Pt(-1920, 0))
	if err := controller.Refresh(profiles.Defaults()); err != nil {
		t.Fatal(err)
	}

	if want := []string{"show", "move"}; !slices.Equal(s.calls, want) || s.at != image.Pt(728-1920, 548) || p.calls != 1 {
		t.Errorf("calls %v at %v after %d paintings, want %v to %v after one", s.calls, s.at, p.calls, want, image.Pt(728-1920, 548))
	}
}

func TestAChangedCrosshairIsRedrawn(t *testing.T) {
	s := &screen{}
	p := &painter{side: 24}
	sc := &scene{target: foreground.Target{Game: game(profiles.Offset{}), Client: hd}, there: true}
	controller := overlay.NewController(s, sc, p.paint)
	if err := controller.Refresh(profiles.Defaults()); err != nil {
		t.Fatal(err)
	}

	sc.target.Game.Crosshair.Color = crosshair.RGB{R: 255}
	if err := controller.Refresh(profiles.Defaults()); err != nil {
		t.Fatal(err)
	}

	if want := []string{"show", "show"}; !slices.Equal(s.calls, want) || p.calls != 2 {
		t.Errorf("calls %v after %d paintings, want %v after two", s.calls, p.calls, want)
	}
}

func TestLeavingTheGameHidesOnceAndComingBackShowsAgain(t *testing.T) {
	s := &screen{}
	p := &painter{side: 24}
	sc := &scene{target: foreground.Target{Game: game(profiles.Offset{}), Client: hd}, there: true}
	controller := overlay.NewController(s, sc, p.paint)

	for _, there := range []bool{true, false, false, true} {
		sc.there = there
		if err := controller.Refresh(profiles.Defaults()); err != nil {
			t.Fatal(err)
		}
	}

	if want := []string{"show", "hide", "show"}; !slices.Equal(s.calls, want) || p.calls != 1 {
		t.Errorf("calls %v after %d paintings, want %v after one", s.calls, p.calls, want)
	}
}

func TestAFailingPaintingHidesTheCrosshairAndIsNotRetriedForTheSameStyle(t *testing.T) {
	s := &screen{}
	p := &painter{side: 24}
	sc := &scene{target: foreground.Target{Game: game(profiles.Offset{}), Client: hd}, there: true}
	controller := overlay.NewController(s, sc, p.paint)
	if err := controller.Refresh(profiles.Defaults()); err != nil {
		t.Fatal(err)
	}

	p.fail = errBroken
	sc.target.Game.Crosshair.Image = "missing.png"
	for i := 0; i < 3; i++ {
		if err := controller.Refresh(profiles.Defaults()); !errors.Is(err, errBroken) {
			t.Fatalf("got %v, want the painting error", err)
		}
	}

	if want := []string{"show", "hide"}; !slices.Equal(s.calls, want) || p.calls != 2 {
		t.Errorf("calls %v after %d paintings, want %v after two", s.calls, p.calls, want)
	}
}

func TestAFailingSceneHidesTheCrosshair(t *testing.T) {
	s := &screen{}
	p := &painter{side: 24}
	sc := &scene{target: foreground.Target{Game: game(profiles.Offset{}), Client: hd}, there: true}
	controller := overlay.NewController(s, sc, p.paint)
	if err := controller.Refresh(profiles.Defaults()); err != nil {
		t.Fatal(err)
	}

	sc.fail = errBroken
	if err := controller.Refresh(profiles.Defaults()); !errors.Is(err, errBroken) {
		t.Fatalf("got %v, want the scene error", err)
	}
	if want := []string{"show", "hide"}; !slices.Equal(s.calls, want) {
		t.Errorf("calls %v, want %v", s.calls, want)
	}
}

func TestAFailedShowIsTriedAgain(t *testing.T) {
	s := &screen{fail: errBroken}
	p := &painter{side: 24}
	controller := overlay.NewController(s, &scene{target: foreground.Target{Game: game(profiles.Offset{}), Client: hd}, there: true}, p.paint)

	if err := controller.Refresh(profiles.Defaults()); !errors.Is(err, errBroken) {
		t.Fatalf("got %v, want the show error", err)
	}

	s.fail = nil
	if err := controller.Refresh(profiles.Defaults()); err != nil {
		t.Fatal(err)
	}
	if want := []string{"show", "hide", "show"}; !slices.Equal(s.calls, want) || s.shown.Pixels == nil {
		t.Errorf("calls %v, want %v: a failed show hides whatever was left, then shows again", s.calls, want)
	}
}
