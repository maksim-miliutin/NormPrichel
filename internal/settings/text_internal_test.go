package settings

import (
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/crosshair"
)

func TestEveryRefusalIsWordedInBothLanguages(t *testing.T) {
	codes := []string{"failed"}
	for _, reason := range reasons {
		codes = append(codes, reason.code)
	}

	for language, words := range texts {
		for _, code := range codes {
			if words["error."+code] == "" {
				t.Errorf("%s: no words for the refusal %q", language, code)
			}
		}
	}
}

func TestEveryShapeIsNamedInBothLanguages(t *testing.T) {
	for language, words := range texts {
		for _, shape := range crosshair.Shapes() {
			if words["shape."+string(shape)] == "" {
				t.Errorf("%s: no name for the shape %q", language, shape)
			}
		}
	}
}

func TestEveryPartIsNamedInBothLanguages(t *testing.T) {
	for language, words := range texts {
		for _, part := range crosshair.PartNames() {
			if words["part."+string(part)] == "" {
				t.Errorf("%s: no name for the part %q", language, part)
			}
			for _, knob := range crosshair.PartKnobs(part) {
				if words[knob.Name] == "" {
					t.Errorf("%s: no label for the %s of the %s", language, knob.Name, part)
				}
			}
		}
	}
}
