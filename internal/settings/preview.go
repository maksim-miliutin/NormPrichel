package settings

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

var (
	darkBackdrop  = color.RGBA{0x22, 0x22, 0x22, 0xFF}
	lightBackdrop = color.RGBA{0xDD, 0xDD, 0xDD, 0xFF}
)

// A picture is named by the hash of its bytes, so a crosshair that paints once paints the same
// until a picture is added: that one may be the very file that went missing.
type painted struct {
	crosshair profiles.Crosshair
	img       *image.RGBA
	err       error
}

func (f *Form) Preview(side int) (*image.RGBA, *image.RGBA, error) {
	game, ok := f.shown(f.State.Current())
	if !ok {
		return bare(side, darkBackdrop), bare(side, lightBackdrop), nil
	}

	img, err := f.paint(game.Crosshair)
	if err != nil {
		return bare(side, darkBackdrop), bare(side, lightBackdrop), err
	}
	fitted := fit(img, side)

	return onto(fitted, side, darkBackdrop), onto(fitted, side, lightBackdrop), nil
}

func (f *Form) paint(c profiles.Crosshair) (*image.RGBA, error) {
	if f.painted != nil && f.painted.crosshair == c {
		return f.painted.img, f.painted.err
	}

	img, err := f.Pictures.Paint(c)
	f.painted = &painted{crosshair: c, img: img, err: err}

	return img, err
}

// A small crosshair grows by whole pixels to about three quarters of the square, so its edges
// stay crisp; a large one shrinks to fit.
func fit(img *image.RGBA, side int) *image.RGBA {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	natural := max(w, h, 1)
	tw, th := w*side/natural, h*side/natural
	if natural <= side {
		zoom := min(max(side*3/4/natural, 1), 4)
		tw, th = w*zoom, h*zoom
	}

	out := image.NewRGBA(image.Rect(0, 0, tw, th))
	for y := 0; y < th; y++ {
		for x := 0; x < tw; x++ {
			out.SetRGBA(x, y, img.RGBAAt(img.Rect.Min.X+x*w/tw, img.Rect.Min.Y+y*h/th))
		}
	}

	return out
}

func onto(img *image.RGBA, side int, backdrop color.RGBA) *image.RGBA {
	out := bare(side, backdrop)
	at := image.Pt((side-img.Rect.Dx())/2, (side-img.Rect.Dy())/2)
	draw.Draw(out, img.Rect.Add(at), img, img.Rect.Min, draw.Over)

	return out
}

func bare(side int, backdrop color.RGBA) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(out, out.Rect, &image.Uniform{C: backdrop}, image.Point{}, draw.Src)

	return out
}
