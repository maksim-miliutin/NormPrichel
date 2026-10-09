package crosshair_test

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
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
	{
		name:  "even ring centred on a pixel corner",
		style: crosshair.Style{Shape: crosshair.Circle, Color: white, Opacity: 100, Radius: 2, Thickness: 2},
		want: []string{
			"++++++",
			"+####+",
			"+#++#+",
			"+#++#+",
			"+####+",
			"++++++",
		},
	},
	{
		name:  "odd ring around a dot, both centred on one pixel",
		style: crosshair.Style{Shape: crosshair.CircleDot, Color: white, Opacity: 100, Radius: 3, Thickness: 3, DotSize: 1},
		want: []string{
			".+++++++.",
			"++#####++",
			"+#######+",
			"+##+++##+",
			"+##+#+##+",
			"+##+++##+",
			"+#######+",
			"++#####++",
			".+++++++.",
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
			if !matches(got, r.want) {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(r.want, "\n"))
			}
		})
	}
}

func TestRingPixelsFollowTheAnnulus(t *testing.T) {
	for _, radius := range []int{2, 3, 5, 10, 64} {
		for _, thickness := range []int{1, 2, 3, 8, 16} {
			style := crosshair.Style{Shape: crosshair.Circle, Color: white, Opacity: 100, Radius: radius, Thickness: thickness}
			img, err := crosshair.Draw(style)
			if err != nil {
				t.Fatal(err)
			}

			if bad := strayPixels(img, annulus{inner: max(0, 2*radius-thickness), outer: 2*radius + thickness}); bad != "" {
				t.Errorf("radius %d thickness %d: %s", radius, thickness, bad)
			}
		}
	}
}

func TestRingAreaMatchesTheAnnulus(t *testing.T) {
	for _, radius := range []int{2, 3, 5, 10, 64} {
		for _, thickness := range []int{1, 2, 3, 8, 16} {
			style := crosshair.Style{Shape: crosshair.Circle, Color: white, Opacity: 100, Radius: radius, Thickness: thickness}
			img, err := crosshair.Draw(style)
			if err != nil {
				t.Fatal(err)
			}

			inner := max(0, float64(radius)-float64(thickness)/2)
			outer := float64(radius) + float64(thickness)/2
			want := math.Pi * (outer*outer - inner*inner)
			if got := ink(img); math.Abs(got-want) > 0.02*want {
				t.Errorf("radius %d thickness %d: %.2f pixels of ink, want %.2f within 2%%", radius, thickness, got, want)
			}
		}
	}
}

func TestCirclesMirrorEightWays(t *testing.T) {
	for _, style := range circles() {
		if style.Shape == crosshair.CircleDot && style.DotSize%2 != style.Thickness%2 {
			continue
		}

		img, err := crosshair.Draw(style)
		if err != nil {
			t.Fatal(err)
		}

		if !same(img, mirrored(img)) || !same(img, transposed(img)) {
			t.Errorf("%+v is not symmetric", style)
		}
	}
}

func circles() []crosshair.Style {
	shapes := []crosshair.Shape{crosshair.Circle, crosshair.CircleDot}
	radii := []int{2, 3, 7, 10}
	const thicknesses, dots = 6, 4

	var out []crosshair.Style
	for i := 0; i < len(shapes)*len(radii)*thicknesses*dots; i++ {
		out = append(out, crosshair.Style{
			Shape:     shapes[i%len(shapes)],
			Color:     white,
			Opacity:   100,
			Radius:    radii[i/len(shapes)%len(radii)],
			Thickness: 1 + i/(len(shapes)*len(radii))%thicknesses,
			DotSize:   1 + i/(len(shapes)*len(radii)*thicknesses)%dots,
		})
	}

	return out
}

func TestRingsSizeTheImageToTheirOuterEdge(t *testing.T) {
	cases := []struct {
		radius, thickness, size int
	}{
		{radius: 10, thickness: 2, size: 22},
		{radius: 10, thickness: 1, size: 21},
		{radius: 100, thickness: 1, size: 129},
		{radius: 0, thickness: 1, size: 5},
	}

	for _, c := range cases {
		style := crosshair.Style{Shape: crosshair.Circle, Color: white, Opacity: 100, Radius: c.radius, Thickness: c.thickness}
		img, err := crosshair.Draw(style)
		if err != nil {
			t.Fatal(err)
		}

		if got := img.Bounds(); got != image.Rect(0, 0, c.size, c.size) {
			t.Errorf("radius %d thickness %d: bounds %v, want a %d pixel square", c.radius, c.thickness, got, c.size)
		}
	}
}

// Radii in half pixels, measured from the image centre.
type annulus struct {
	inner, outer int
}

// strayPixels names the first pixel that lies wholly inside the band yet is not opaque, or
// wholly outside it yet holds any ink. Everything counts in half pixels, so it stays exact.
func strayPixels(img *image.RGBA, ring annulus) string {
	centre := img.Bounds().Dx()
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			near, far := reach(2*x-centre, 2*y-centre)
			alpha := img.RGBAAt(x, y).A
			inside := near >= ring.inner*ring.inner && far <= ring.outer*ring.outer
			outside := near > ring.outer*ring.outer || far < ring.inner*ring.inner
			if inside && alpha != 255 || outside && alpha != 0 {
				return fmt.Sprintf("pixel %d,%d has alpha %d", x, y, alpha)
			}
		}
	}

	return ""
}

// reach gives the squared distances from the centre to the nearest and farthest points of
// the two-unit square whose top left corner sits at x, y.
func reach(x, y int) (near, far int) {
	nx, ny := max(x, 0, -x-2), max(y, 0, -y-2)
	fx, fy := max(abs(x), abs(x+2)), max(abs(y), abs(y+2))

	return nx*nx + ny*ny, fx*fx + fy*fy
}

func abs(v int) int {
	return max(v, -v)
}

func ink(img *image.RGBA) float64 {
	total := 0.0
	for i := 3; i < len(img.Pix); i += 4 {
		total += float64(img.Pix[i]) / 255
	}

	return total
}

// A '+' in a reference marks a pixel that the edge of a ring crosses: any coverage of the fill
// colour passes there, but nothing else.
func matches(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}

	for y := range want {
		if !rowMatches(got[y], want[y]) {
			return false
		}
	}

	return true
}

func rowMatches(got, want string) bool {
	if len(got) != len(want) {
		return false
	}

	for x := range want {
		if !pixelMatches(got[x], want[x]) {
			return false
		}
	}

	return true
}

func pixelMatches(got, want byte) bool {
	if want == '+' {
		return strings.IndexByte(".+#", got) >= 0
	}

	return got == want
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
			row.WriteByte(glyph(img.RGBAAt(x, y)))
		}
		rows = append(rows, row.String())
	}

	return rows
}

func glyph(c color.RGBA) byte {
	switch {
	case c == color.RGBA{}:
		return '.'
	case c == color.RGBA{R: 255, G: 255, B: 255, A: 255}:
		return '#'
	case c.R == c.A && c.G == c.A && c.B == c.A:
		return '+'
	}

	return '?'
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
