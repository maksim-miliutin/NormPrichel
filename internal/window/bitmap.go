package window

import "image"

// A device-independent bitmap of 32 bits takes each pixel as blue, green, red and a byte unused.
func bgra(img *image.RGBA) []byte {
	out := make([]byte, len(img.Pix))
	for i := 0; i+3 < len(img.Pix); i += 4 {
		out[i], out[i+1], out[i+2] = img.Pix[i+2], img.Pix[i+1], img.Pix[i]
	}

	return out
}
