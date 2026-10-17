package window

import (
	"image"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

// Virtual-key codes from the Windows headers.
const (
	keyPageUp   = 0x21
	keyPageDown = 0x22
	keyEnd      = 0x23
	keyHome     = 0x24
	keyLeft     = 0x25
	keyUp       = 0x26
	keyRight    = 0x27
	keyDown     = 0x28
)

type Track struct {
	Width, Thumb int // device pixels
}

// The thumb's room is as wide as the ring it wears with the keys, so the ring stays whole at either end.
func NewTrack(width, dpi int) Track {
	return Track{Width: width, Thumb: scaled(24, dpi)}
}

func (t Track) Image(height, at int, keyed bool) *image.RGBA {
	middle, centre, rail := height/2, at+t.Thumb/2, max(t.Thumb/6, 1)
	disc := func(diameter int, colour crosshair.RGB) Layer {
		corner := image.Pt(centre-diameter/2, middle-diameter/2)
		return Layer{image.Rectangle{Min: corner, Max: corner.Add(image.Pt(diameter, diameter))}, pill, colour}
	}
	groove := image.Rect(t.Thumb/2, middle-rail/2, t.Width-t.Thumb/2, middle-rail/2+rail)
	filled := groove
	filled.Max.X = centre
	layers := []Layer{{groove, pill, seam}, {filled, pill, signal}}
	if keyed {
		layers = append(layers, disc(t.Thumb, ink), disc(t.Thumb-2*max(t.Thumb/12, 1), ground))
	}
	layers = append(layers, disc(t.Thumb*2/3, ink), disc(t.Thumb/3, signal))

	return Layered(image.Pt(t.Width, height), ground, layers...)
}

func (t Track) ThumbAt(knob crosshair.Knob, value int) int {
	span, steps := t.Width-t.Thumb, stepsOf(knob)
	if span <= 0 || steps == 0 {
		return 0
	}
	step := (min(max(value, knob.Min), knob.Max) - knob.Min) / knob.Step

	return (step*span*2 + steps) / (steps * 2)
}

func (t Track) ValueAt(knob crosshair.Knob, x int) int {
	span, steps := t.Width-t.Thumb, stepsOf(knob)
	if span <= 0 || steps == 0 {
		return knob.Min
	}
	along := min(max(x-t.Thumb/2, 0), span)
	step := (along*steps*2 + span) / (span * 2)

	return min(knob.Min+step*knob.Step, knob.Max)
}

func Keyed(knob crosshair.Knob, value int, key uint32) (int, bool) {
	moves := map[uint32]int{keyLeft: -1, keyDown: -1, keyRight: 1, keyUp: 1, keyPageDown: -10, keyPageUp: 10}
	switch key {
	case keyHome:
		return knob.Min, true
	case keyEnd:
		return knob.Max, true
	}
	move, ok := moves[key]
	if !ok {
		return value, false
	}

	return min(max(value+move*knob.Step, knob.Min), knob.Max), true
}

func stepsOf(knob crosshair.Knob) int {
	if knob.Step <= 0 {
		return 0
	}

	return (knob.Max - knob.Min) / knob.Step
}
