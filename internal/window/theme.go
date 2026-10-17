package window

import "github.com/maksim-miliutin/NormPrichel/internal/crosshair"

// Warm graphite rather than black and white; one amber, the glow of a scope or a PDA in the Zone,
// marks whatever is chosen or switched on.
var (
	ground   = crosshair.RGB{R: 0x1B, G: 0x1D, B: 0x1A}
	field    = crosshair.RGB{R: 0x28, G: 0x2B, B: 0x25}
	seam     = crosshair.RGB{R: 0x43, G: 0x47, B: 0x3E}
	ink      = crosshair.RGB{R: 0xE9, G: 0xE6, B: 0xDC}
	ash      = crosshair.RGB{R: 0xA6, G: 0xA7, B: 0x9B}
	signal   = crosshair.RGB{R: 0xE5, G: 0xA1, B: 0x3A}
	onSignal = ground
	rust     = crosshair.RGB{R: 0xA8, G: 0x43, B: 0x2F}
)
