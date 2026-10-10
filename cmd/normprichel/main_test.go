package main

import (
	"errors"
	"strings"
	"testing"
)

func TestAnErrorIsPrintedOnceUntilSomethingElseHappens(t *testing.T) {
	var out strings.Builder
	report := reporter(&out)

	first, second := errors.New("first"), errors.New("second")
	for _, err := range []error{first, first, first, nil, first, second, second} {
		report(err)
	}

	if want := "first\nfirst\nsecond\n"; out.String() != want {
		t.Errorf("printed %q, want %q", out.String(), want)
	}
}
