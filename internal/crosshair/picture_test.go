package crosshair_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

var (
	hiddenRed   = color.NRGBA{R: 255, A: 0}
	solid       = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	opaqueWhite = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

func TestAPictureAtFullScaleKeepsEveryPixel(t *testing.T) {
	file := encoded(t, picture(3, 1, []color.NRGBA{
		{R: 255, G: 0, B: 0, A: 128},
		{R: 10, G: 20, B: 30, A: 255},
		{R: 255, G: 255, B: 255, A: 0},
	}))

	img, err := crosshair.FromPNG(file, 100)
	if err != nil {
		t.Fatal(err)
	}

	want := []color.RGBA{{R: 128, A: 128}, {R: 10, G: 20, B: 30, A: 255}, {}}
	for x, w := range want {
		if got := img.RGBAAt(x, 0); got != w {
			t.Errorf("pixel %d is %v, want %v", x, got, w)
		}
	}
}

func TestDoublingBlendsNeighboursByQuarters(t *testing.T) {
	img, err := crosshair.FromPNG(encoded(t, picture(2, 1, []color.NRGBA{solid, hiddenRed})), 200)
	if err != nil {
		t.Fatal(err)
	}

	if got := img.Bounds(); got != image.Rect(0, 0, 4, 2) {
		t.Fatalf("bounds %v, want 4 by 2", got)
	}

	for y := 0; y < 2; y++ {
		for x, v := range []uint8{255, 191, 64, 0} {
			if got, want := img.RGBAAt(x, y), (color.RGBA{R: v, G: v, B: v, A: v}); got != want {
				t.Errorf("pixel %d,%d is %v, want %v", x, y, got, want)
			}
		}
	}
}

func TestQuarterScaleAveragesTheWholeNeighbourhood(t *testing.T) {
	var board []color.NRGBA
	for i := 0; i < 16; i++ {
		board = append(board, []color.NRGBA{solid, hiddenRed}[(i%4+i/4)%2])
	}

	img, err := crosshair.FromPNG(encoded(t, picture(4, 4, board)), 25)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := img.RGBAAt(0, 0), (color.RGBA{R: 128, G: 128, B: 128, A: 128}); got != want || img.Bounds() != image.Rect(0, 0, 1, 1) {
		t.Errorf("got %v over %v, want %v over one pixel", got, img.Bounds(), want)
	}
}

func TestShrinkingFadesAThinLineInsteadOfLosingIt(t *testing.T) {
	pixels := make([]color.NRGBA, 64)
	for i := range pixels {
		pixels[i] = hiddenRed
		if i%8 == 0 {
			pixels[i] = solid
		}
	}

	img, err := crosshair.FromPNG(encoded(t, picture(8, 8, pixels)), 25)
	if err != nil {
		t.Fatal(err)
	}

	for y := 0; y < 2; y++ {
		if got, want := img.RGBAAt(0, y), (color.RGBA{R: 46, G: 46, B: 46, A: 46}); got != want {
			t.Errorf("left pixel %d is %v, want the line faded to %v", y, got, want)
		}
		if got := img.RGBAAt(1, y); got != (color.RGBA{}) {
			t.Errorf("right pixel %d is %v, want it clear of a line four pixels away", y, got)
		}
	}
}

func TestScalingNeverTintsTheEdges(t *testing.T) {
	var pixels []color.NRGBA
	for i := 0; i < 64; i++ {
		x, y := i%8, i/8
		if x >= 2 && x < 6 && y >= 2 && y < 6 {
			pixels = append(pixels, solid)
			continue
		}
		pixels = append(pixels, hiddenRed)
	}
	file := encoded(t, picture(8, 8, pixels))

	for scale := 25; scale <= 400; scale += 25 {
		img, err := crosshair.FromPNG(file, scale)
		if err != nil {
			t.Fatal(err)
		}

		for i := 0; i < len(img.Pix); i += 4 {
			if r, g, b, a := img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]; r != a || g != a || b != a {
				t.Fatalf("scale %d: pixel %d is %d,%d,%d,%d, want white with its own alpha", scale, i/4, r, g, b, a)
			}
		}
	}
}

func TestScalingKeepsThePictureCentred(t *testing.T) {
	for _, side := range []int{3, 4, 5} {
		file := encoded(t, picture(side, side, plus(side)))

		for scale := 25; scale <= 400; scale += 25 {
			img, err := crosshair.FromPNG(file, scale)
			if err != nil {
				t.Fatal(err)
			}

			if !same(img, mirrored(img)) || !same(img, transposed(img)) {
				t.Errorf("side %d at %d%% is no longer symmetric", side, scale)
			}
		}
	}
}

func TestScaleRoundsTheSizeAndStaysWithin25To400(t *testing.T) {
	cases := []struct {
		side, scale, want int
	}{
		{side: 3, scale: 50, want: 2},
		{side: 63, scale: 25, want: 16},
		{side: 3, scale: 25, want: 1},
		{side: 8, scale: 10, want: 2},
		{side: 2, scale: 1000, want: 8},
	}

	for _, c := range cases {
		pixels := make([]color.NRGBA, c.side*c.side)
		for i := range pixels {
			pixels[i] = solid
		}

		img, err := crosshair.FromPNG(encoded(t, picture(c.side, c.side, pixels)), c.scale)
		if err != nil {
			t.Fatal(err)
		}

		if got := img.Bounds(); got != image.Rect(0, 0, c.want, c.want) {
			t.Errorf("side %d at %d%%: bounds %v, want %d square", c.side, c.scale, got, c.want)
		}
		if got := img.RGBAAt(0, 0); got != opaqueWhite {
			t.Errorf("side %d at %d%%: a solid picture became %v", c.side, c.scale, got)
		}
	}
}

func TestAFileOver2MBIsRefusedUnread(t *testing.T) {
	_, err := crosshair.FromPNG(make([]byte, 2<<20+1), 100)
	if !errors.Is(err, crosshair.ErrFileTooBig) {
		t.Errorf("one byte over: got %v, want ErrFileTooBig", err)
	}

	_, err = crosshair.FromPNG(make([]byte, 2<<20), 100)
	if !errors.Is(err, crosshair.ErrNotPNG) {
		t.Errorf("exactly 2 MB of zeros: got %v, want ErrNotPNG", err)
	}
}

func TestAPictureOver512IsRefusedFromItsHeader(t *testing.T) {
	cases := []struct {
		name string
		file []byte
		want error
	}{
		{name: "a header claiming 100000 pixels with no image data", file: header(100000, 100000), want: crosshair.ErrTooLarge},
		{name: "a real picture one pixel too wide", file: encoded(t, picture(513, 1, make([]color.NRGBA, 513))), want: crosshair.ErrTooLarge},
		{name: "a real picture one pixel too tall", file: encoded(t, picture(1, 513, make([]color.NRGBA, 513))), want: crosshair.ErrTooLarge},
		{name: "the largest allowed picture", file: encoded(t, picture(512, 512, make([]color.NRGBA, 512*512))), want: nil},
	}

	for _, c := range cases {
		if _, err := crosshair.FromPNG(c.file, 100); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}

func TestBrokenFilesAreRefused(t *testing.T) {
	whole := encoded(t, picture(2, 2, []color.NRGBA{solid, hiddenRed, hiddenRed, solid}))
	cases := map[string][]byte{
		"empty":           nil,
		"text":            []byte("not a picture"),
		"cut in the data": whole[:len(whole)-20],
	}

	for name, file := range cases {
		if _, err := crosshair.FromPNG(file, 100); !errors.Is(err, crosshair.ErrNotPNG) {
			t.Errorf("%s: got %v, want ErrNotPNG", name, err)
		}
	}
}

// plus draws a plus through the middle of the square: one pixel wide when the side is odd,
// two when it is even, so the picture is symmetric about its centre either way.
func plus(side int) []color.NRGBA {
	middle := map[int]bool{side / 2: true, (side - 1) / 2: true}
	pixels := make([]color.NRGBA, side*side)
	for i := range pixels {
		pixels[i] = hiddenRed
		if middle[i%side] || middle[i/side] {
			pixels[i] = solid
		}
	}

	return pixels
}

func picture(width, height int, pixels []color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for i, c := range pixels {
		img.SetNRGBA(i%width, i/width, c)
	}

	return img
}

func encoded(t *testing.T, img image.Image) []byte {
	t.Helper()

	var file bytes.Buffer
	if err := png.Encode(&file, img); err != nil {
		t.Fatal(err)
	}

	return file.Bytes()
}

// header writes a PNG signature, an IHDR chunk for an 8-bit RGBA picture of the given size
// and IEND, with no image data between them.
func header(width, height uint32) []byte {
	var file bytes.Buffer
	file.WriteString("\x89PNG\r\n\x1a\n")

	ihdr := binary.BigEndian.AppendUint32(nil, width)
	ihdr = binary.BigEndian.AppendUint32(ihdr, height)
	ihdr = append(ihdr, 8, 6, 0, 0, 0)
	file.Write(chunk("IHDR", ihdr))
	file.Write(chunk("IEND", nil))

	return file.Bytes()
}

func chunk(kind string, body []byte) []byte {
	out := binary.BigEndian.AppendUint32(nil, uint32(len(body)))
	out = append(out, kind...)
	out = append(out, body...)

	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(append([]byte(kind), body...)))
}
