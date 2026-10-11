package settings

import "testing"

func TestEveryRefusalIsWordedInBothLanguages(t *testing.T) {
	codes := []string{"failed", "offline", "stale"}
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
