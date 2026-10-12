package window

import (
	"image"
	"image/color"
	"slices"
	"testing"
)

func TestAPictureGoesToWindowsBlueFirst(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.SetRGBA(0, 0, color.RGBA{R: 0xFF, G: 0x80, B: 0x10, A: 0xFF})
	img.SetRGBA(1, 0, color.RGBA{R: 0x01, G: 0x02, B: 0x03, A: 0xFF})

	if got, want := bgra(img), []byte{0x10, 0x80, 0xFF, 0, 0x03, 0x02, 0x01, 0}; !slices.Equal(got, want) {
		t.Errorf("got % x, want % x", got, want)
	}
}
