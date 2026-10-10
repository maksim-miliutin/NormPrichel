package overlay

import (
	"errors"
	"image"

	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

// What UpdateLayeredWindow takes: premultiplied blue, green, red and alpha, rows from the top.
type Bitmap struct {
	Pixels []byte
	Size   image.Point
}

type Screen interface {
	Show(bitmap Bitmap, at image.Point) error
	Move(at image.Point) error
	Hide() error
}

type Targets interface {
	Target(settings profiles.Settings) (foreground.Target, bool, error)
}

type Painter func(crosshair profiles.Crosshair) (*image.RGBA, error)

// Controller keeps the overlay in line with the window in front and touches the screen only
// for what changed: a game that redraws every frame must not pay for a crosshair standing still.
type Controller struct {
	screen  Screen
	targets Targets
	paint   Painter

	ready   bool
	painted profiles.Crosshair
	bitmap  *Bitmap
	failure error

	onScreen *Bitmap
	at       image.Point
}

func NewController(screen Screen, targets Targets, paint Painter) *Controller {
	return &Controller{screen: screen, targets: targets, paint: paint}
}

func (c *Controller) Refresh(settings profiles.Settings) error {
	target, ok, err := c.targets.Target(settings)
	if err != nil || !ok {
		return errors.Join(err, c.hide())
	}

	bitmap, err := c.bitmapFor(target.Game.Crosshair)
	if err != nil {
		return errors.Join(err, c.hide())
	}

	return c.put(bitmap, placed(target, bitmap.Size))
}

func (c *Controller) bitmapFor(crosshair profiles.Crosshair) (*Bitmap, error) {
	if c.ready && crosshair == c.painted {
		return c.bitmap, c.failure
	}

	img, err := c.paint(crosshair)
	c.ready, c.painted, c.bitmap, c.failure = true, crosshair, nil, err
	if err == nil {
		c.bitmap = bitmapOf(img)
	}

	return c.bitmap, c.failure
}

func (c *Controller) put(bitmap *Bitmap, at image.Point) error {
	if c.onScreen == bitmap && c.at == at {
		return nil
	}

	if c.onScreen == bitmap {
		if err := c.screen.Move(at); err != nil {
			return err
		}
		c.at = at

		return nil
	}

	if err := c.screen.Show(*bitmap, at); err != nil {
		c.onScreen = nil

		return errors.Join(err, c.screen.Hide())
	}
	c.onScreen, c.at = bitmap, at

	return nil
}

func (c *Controller) hide() error {
	if c.onScreen == nil {
		return nil
	}
	c.onScreen = nil

	return c.screen.Hide()
}

// The pixel at half the bitmap's size goes on the pixel at half the client area's size.
func placed(target foreground.Target, size image.Point) image.Point {
	centre := target.Client.Min.Add(target.Client.Size().Div(2))
	offset := image.Pt(target.Game.Offset.X, target.Game.Offset.Y)

	return centre.Add(offset).Sub(size.Div(2))
}

func bitmapOf(img *image.RGBA) *Bitmap {
	size := img.Rect.Size()
	pixels := make([]byte, 0, 4*size.X*size.Y)
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		row := img.Pix[img.PixOffset(img.Rect.Min.X, y):][:4*size.X]
		for i := 0; i < len(row); i += 4 {
			pixels = append(pixels, row[i+2], row[i+1], row[i], row[i+3])
		}
	}

	return &Bitmap{Pixels: pixels, Size: size}
}
