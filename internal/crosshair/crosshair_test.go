package crosshair_test

import (
	"errors"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

var white = crosshair.RGB{R: 255, G: 255, B: 255}

type reference struct {
	name  string
	style crosshair.Style
	want  []string
}

var references = []reference{
	{
		name:  "odd thickness centres the arms on one pixel",
		style: crosshair.Style{Shape: crosshair.Cross, Color: white, Opacity: 100, Length: 2, Thickness: 1, Gap: 1},
		want: []string{
			"..#..",
			"..#..",
			"##.##",
			"..#..",
			"..#..",
		},
	},
	{
		name:  "even thickness centres the arms on a pixel corner",
		style: crosshair.Style{Shape: crosshair.Cross, Color: white, Opacity: 100, Length: 2, Thickness: 2, Gap: 1},
		want: []string{
			"..##..",
			"..##..",
			"##..##",
			"##..##",
			"..##..",
			"..##..",
		},
	},
	{
		name:  "zero gap with odd thickness leaves no hole in the middle",
		style: crosshair.Style{Shape: crosshair.Cross, Color: white, Opacity: 100, Length: 2, Thickness: 1, Gap: 0},
		want: []string{
			".#.",
			"###",
			".#.",
		},
	},
	{
		name:  "zero gap with even thickness",
		style: crosshair.Style{Shape: crosshair.Cross, Color: white, Opacity: 100, Length: 2, Thickness: 2, Gap: 0},
		want: []string{
			".##.",
			"####",
			"####",
			".##.",
		},
	},
	{
		name:  "tee drops the upper arm and keeps the cross's frame",
		style: crosshair.Style{Shape: crosshair.Tee, Color: white, Opacity: 100, Length: 2, Thickness: 1, Gap: 1},
		want: []string{
			".....",
			".....",
			"##.##",
			"..#..",
			"..#..",
		},
	},
	{
		name:  "one pixel dot",
		style: crosshair.Style{Shape: crosshair.Dot, Color: white, Opacity: 100, DotSize: 1},
		want: []string{
			"#",
		},
	},
	{
		name:  "even dot",
		style: crosshair.Style{Shape: crosshair.Dot, Color: white, Opacity: 100, DotSize: 2},
		want: []string{
			"##",
			"##",
		},
	},
	{
		name:  "odd dot",
		style: crosshair.Style{Shape: crosshair.Dot, Color: white, Opacity: 100, DotSize: 3},
		want: []string{
			"###",
			"###",
			"###",
		},
	},
	{
		name:  "dot inside the gap of a cross of the same parity",
		style: crosshair.Style{Shape: crosshair.CrossDot, Color: white, Opacity: 100, Length: 2, Thickness: 1, Gap: 2, DotSize: 1},
		want: []string{
			"...#...",
			"...#...",
			".......",
			"##.#.##",
			".......",
			"...#...",
			"...#...",
		},
	},
	{
		name:  "odd dot in an even cross leans right and down",
		style: crosshair.Style{Shape: crosshair.CrossDot, Color: white, Opacity: 100, Length: 1, Thickness: 2, Gap: 2, DotSize: 1},
		want: []string{
			"..##..",
			"......",
			"#....#",
			"#..#.#",
			"......",
			"..##..",
		},
	},
	{
		name:  "even dot in an odd cross leans right and down",
		style: crosshair.Style{Shape: crosshair.CrossDot, Color: white, Opacity: 100, Length: 1, Thickness: 1, Gap: 2, DotSize: 2},
		want: []string{
			"..#..",
			".....",
			"#.###",
			"..##.",
			"..#..",
		},
	},
	{
		name:  "a dot wider than the cross widens the image on its far side",
		style: crosshair.Style{Shape: crosshair.CrossDot, Color: white, Opacity: 100, Length: 1, Thickness: 2, Gap: 0, DotSize: 3},
		want: []string{
			"....",
			".###",
			".###",
			".###",
		},
	},
}

func TestShapesMatchHandDrawnReferences(t *testing.T) {
	for _, r := range references {
		t.Run(r.name, func(t *testing.T) {
			img, err := crosshair.Draw(r.style)
			if err != nil {
				t.Fatal(err)
			}

			got := art(img)
			if strings.Join(got, "\n") != strings.Join(r.want, "\n") {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(r.want, "\n"))
			}
		})
	}
}

func TestEveryShapeHasAReference(t *testing.T) {
	drawn := map[crosshair.Shape]bool{}
	for _, r := range references {
		drawn[r.style.Shape] = true
	}

	for _, shape := range crosshair.Shapes() {
		if !drawn[shape] {
			t.Errorf("no reference drawing for shape %q; add one with the shape", shape)
		}
	}
}

func TestOpacityScalesColourAndAlphaAlike(t *testing.T) {
	cases := []struct {
		opacity int
		want    color.RGBA
	}{
		{opacity: 100, want: color.RGBA{R: 0x00, G: 0xFF, B: 0x66, A: 255}},
		{opacity: 50, want: color.RGBA{R: 0, G: 128, B: 51, A: 128}},
		{opacity: 10, want: color.RGBA{R: 0, G: 26, B: 10, A: 26}},
	}

	for _, c := range cases {
		style := crosshair.Style{Shape: crosshair.Dot, Color: crosshair.RGB{R: 0x00, G: 0xFF, B: 0x66}, Opacity: c.opacity, DotSize: 1}
		img, err := crosshair.Draw(style)
		if err != nil {
			t.Fatal(err)
		}

		if got := img.RGBAAt(0, 0); got != c.want {
			t.Errorf("opacity %d: got %v, want %v", c.opacity, got, c.want)
		}
	}
}

func TestOutOfRangeValuesDrawAsTheNearestLimit(t *testing.T) {
	style := crosshair.Style{Shape: crosshair.Cross, Color: white, Opacity: 0, Length: 1000, Thickness: 99, Gap: -5}
	img, err := crosshair.Draw(style)
	if err != nil {
		t.Fatal(err)
	}

	if got := img.Bounds(); got != image.Rect(0, 0, 128, 128) {
		t.Fatalf("bounds %v, want a 128 pixel square for length 64 and no gap", got)
	}

	if got := img.RGBAAt(64, 64).A; got != 26 {
		t.Errorf("centre alpha %d, want 26 for the 10 percent floor and a filled centre", got)
	}

	thick := 0
	for y := 0; y < 128; y++ {
		if img.RGBAAt(127, y).A != 0 {
			thick++
		}
	}
	if thick != 16 {
		t.Errorf("arm is %d pixels thick, want the 16 pixel ceiling", thick)
	}

	tiny, err := crosshair.Draw(crosshair.Style{Shape: crosshair.Dot, Color: white, Opacity: 100, DotSize: 0})
	if err != nil {
		t.Fatal(err)
	}
	if got := tiny.Bounds(); got != image.Rect(0, 0, 1, 1) {
		t.Errorf("dot of size 0 drew %v, want the one pixel floor", got)
	}
}

func TestAnUnknownShapeIsRefused(t *testing.T) {
	_, err := crosshair.Draw(crosshair.Style{Shape: "star", Color: white, Opacity: 100})
	if !errors.Is(err, crosshair.ErrUnknownShape) {
		t.Fatalf("got %v, want ErrUnknownShape", err)
	}
}

func TestSymmetricShapesMirrorAcrossTheCentre(t *testing.T) {
	for _, shape := range []crosshair.Shape{crosshair.Cross, crosshair.Dot, crosshair.CrossDot, crosshair.Tee} {
		for _, style := range styles(shape) {
			if shape == crosshair.CrossDot && style.DotSize%2 != style.Thickness%2 {
				continue
			}

			img, err := crosshair.Draw(style)
			if err != nil {
				t.Fatal(err)
			}

			if !same(img, mirrored(img)) {
				t.Errorf("%+v is not symmetric left to right", style)
			}
			if shape != crosshair.Tee && !same(img, transposed(img)) {
				t.Errorf("%+v is not symmetric along the diagonal", style)
			}
		}
	}
}

func TestArmsClearOfEachOtherHoldLengthTimesThicknessPixels(t *testing.T) {
	arms := map[crosshair.Shape]int{crosshair.Cross: 4, crosshair.Tee: 3}
	for shape, count := range arms {
		for _, style := range styles(shape) {
			if style.Gap < (style.Thickness+1)/2 {
				continue
			}

			img, err := crosshair.Draw(style)
			if err != nil {
				t.Fatal(err)
			}

			if got, want := opaque(img), count*style.Length*style.Thickness; got != want {
				t.Errorf("%+v holds %d pixels, want %d", style, got, want)
			}
		}
	}
}

func styles(shape crosshair.Shape) []crosshair.Style {
	const thicknesses, gaps, lengths, dots = 6, 4, 5, 4

	var out []crosshair.Style
	for i := 0; i < thicknesses*gaps*lengths*dots; i++ {
		out = append(out, crosshair.Style{
			Shape:     shape,
			Color:     white,
			Opacity:   100,
			Thickness: 1 + i%thicknesses,
			Gap:       i / thicknesses % gaps,
			Length:    1 + i/(thicknesses*gaps)%lengths,
			DotSize:   1 + i/(thicknesses*gaps*lengths)%dots,
		})
	}

	return out
}

func art(img *image.RGBA) []string {
	var rows []string
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		var row strings.Builder
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			switch img.RGBAAt(x, y) {
			case color.RGBA{}:
				row.WriteByte('.')
			case color.RGBA{R: 255, G: 255, B: 255, A: 255}:
				row.WriteByte('#')
			default:
				row.WriteByte('?')
			}
		}
		rows = append(rows, row.String())
	}

	return rows
}

func mirrored(img *image.RGBA) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.SetRGBA(b.Max.X-1-(x-b.Min.X), y, img.RGBAAt(x, y))
		}
	}

	return out
}

func transposed(img *image.RGBA) *image.RGBA {
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(b.Min.Y, b.Min.X, b.Max.Y, b.Max.X))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.SetRGBA(y, x, img.RGBAAt(x, y))
		}
	}

	return out
}

func same(a, b *image.RGBA) bool {
	return a.Bounds() == b.Bounds() && string(a.Pix) == string(b.Pix)
}

func opaque(img *image.RGBA) int {
	count := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0 {
			count++
		}
	}

	return count
}
