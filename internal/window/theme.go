package window

import "github.com/maksim-miliutin/NormPrichel/internal/crosshair"

// Warm graphite rather than black and white; one amber, the glow of a scope or a PDA in the Zone,
// marks whatever is chosen or switched on.
var (
	ground   = crosshair.RGB{R: 0x1B, G: 0x1D, B: 0x1A}
	field    = crosshair.RGB{R: 0x28, G: 0x2B, B: 0x25}
	lift     = crosshair.RGB{R: 0x33, G: 0x37, B: 0x2F}
	seam     = crosshair.RGB{R: 0x43, G: 0x47, B: 0x3E}
	rim      = crosshair.RGB{R: 0x4E, G: 0x53, B: 0x48}
	ink      = crosshair.RGB{R: 0xE9, G: 0xE6, B: 0xDC}
	ash      = crosshair.RGB{R: 0xA6, G: 0xA7, B: 0x9B}
	signal   = crosshair.RGB{R: 0xE5, G: 0xA1, B: 0x3A}
	onSignal = ground
	rust     = crosshair.RGB{R: 0xA8, G: 0x43, B: 0x2F}
)

// The pointer finds what it can press by a lighter shade; amber stays the mark of a choice made,
// so a chosen button keeps it under the mouse too.
func face(chosen, hovered bool) (words, paper, edge crosshair.RGB) {
	switch {
	case chosen:
		return onSignal, signal, signal
	case hovered:
		return ink, lift, rim
	}

	return ink, field, seam
}

func tabInk(open, hovered bool) crosshair.RGB {
	if open || hovered {
		return ink
	}

	return ash
}
