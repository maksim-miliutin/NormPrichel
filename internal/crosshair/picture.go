package crosshair

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
)

var (
	ErrFileTooBig = errors.New("crosshair: the file is over 2 MB")
	ErrNotPNG     = errors.New("crosshair: not a PNG picture")
	ErrTooLarge   = errors.New("crosshair: the picture is over 512 pixels on a side")
)

const (
	maxFile = 2 << 20 // bytes
	maxSide = 512
)

var scaleLimit = limit{min: 25, max: 400}

func FromPNG(file []byte, scale int) (*image.RGBA, error) {
	if len(file) > maxFile {
		return nil, ErrFileTooBig
	}

	// The header goes first: a small file can claim a huge picture, and Decode would allocate it.
	config, err := png.DecodeConfig(bytes.NewReader(file))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotPNG, err)
	}
	if config.Width > maxSide || config.Height > maxSide {
		return nil, fmt.Errorf("%w: %d by %d", ErrTooLarge, config.Width, config.Height)
	}

	decoded, err := png.Decode(bytes.NewReader(file))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotPNG, err)
	}

	source := image.NewRGBA(image.Rect(0, 0, config.Width, config.Height))
	draw.Draw(source, source.Rect, decoded, decoded.Bounds().Min, draw.Src)

	return resized(source, scaleLimit.clamp(scale)), nil
}

// resized filters premultiplied pixels with a tent as wide as a source pixel when enlarging and
// as wide as an output pixel when shrinking, so thin lines fade instead of vanishing.
func resized(src *image.RGBA, percent int) *image.RGBA {
	across := filters(src.Rect.Dx(), percent)
	down := filters(src.Rect.Dy(), percent)

	wide := make([]int32, len(across)*src.Rect.Dy()*4)
	for i := range wide {
		channel, x, y := i%4, i/4%len(across), i/4/len(across)
		wide[i] = int32(across[x].sum(func(from int) int { return int(src.Pix[y*src.Stride+4*from+channel]) }))
	}

	out := image.NewRGBA(image.Rect(0, 0, len(across), len(down)))
	for i := range out.Pix {
		channel, x, y := i%4, i/4%len(across), i/4/len(across)
		column := down[y].sum(func(from int) int { return int(wide[(from*len(across)+x)*4+channel]) })
		out.Pix[i] = uint8(roundedRatio(column, across[x].total*down[y].total))
	}

	return out
}

type filter struct {
	first   int
	weights []int
	total   int
}

// Positions count in 1/(2*percent) of a source pixel, which keeps every centre and reach whole.
func filters(in, percent int) []filter {
	out := max(1, roundedRatio(in*percent, 100))
	reach := max(2*percent, 200)

	fs := make([]filter, out)
	for o := range fs {
		centre := (2*o+1-out)*100 + (in-1)*percent
		for i := 0; i < in; i++ {
			fs[o] = fs[o].with(i, reach-abs(2*percent*i-centre))
		}
	}

	return fs
}

func (f filter) with(from, weight int) filter {
	if weight <= 0 {
		return f
	}
	if f.weights == nil {
		f.first = from
	}

	return filter{first: f.first, weights: append(f.weights, weight), total: f.total + weight}
}

func (f filter) sum(value func(from int) int) int {
	total := 0
	for k, w := range f.weights {
		total += w * value(f.first+k)
	}

	return total
}

func abs(v int) int {
	return max(v, -v)
}
