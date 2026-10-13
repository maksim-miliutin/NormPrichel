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
