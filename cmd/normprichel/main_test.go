package main

import (
	"errors"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/maksim-miliutin/NormPrichel/internal/tray"
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

func TestATaggedBuildShowsItsTagAndAnyOtherSaysDev(t *testing.T) {
	built := func(version string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Version: version}}
	}

	cases := []struct {
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{info: built("v0.1.0"), ok: true, want: "v0.1.0"},
		{info: built("v0.1.0+dirty"), ok: true, want: "v0.1.0+dirty"},
		{info: built("v0.0.0-20261011141603-abcdef123456+dirty"), ok: true, want: "dev"},
		{info: built("v0.1.1-0.20261011141603-abcdef123456"), ok: true, want: "dev"},
		{info: built("(devel)"), ok: true, want: "dev"},
		{info: built(""), ok: true, want: "dev"},
		{info: nil, ok: false, want: "dev"},
	}

	for _, c := range cases {
		if got := versionOf(c.info, c.ok); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
}

func TestEveryStartByHandShowsTheSettingsAndASignInStartDoesNot(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{args: nil, want: true},
		{args: []string{"--verbose"}, want: true},
		{args: []string{tray.Background}, want: false},
		{args: []string{"--verbose", tray.Background}, want: false},
	}

	for _, c := range cases {
		if got := settingsAtStart(c.args); got != c.want {
			t.Errorf("args %q: got %v, want %v", c.args, got, c.want)
		}
	}
}
