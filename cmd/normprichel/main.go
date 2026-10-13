package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime/debug"
	"slices"

	"github.com/maksim-miliutin/NormPrichel/internal/tray"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// The overlay looks at the game ten times a second, and a lasting fault would fill the console;
// each fault is printed once, and again only after something else happened in between.
func reporter(out io.Writer) func(error) {
	last := ""

	return func(err error) {
		text := ""
		if err != nil {
			text = err.Error()
		}
		if text != "" && text != last {
			fmt.Fprintln(out, text)
		}
		last = text
	}
}

// A build from a tagged commit carries the tag as its version. Go stamps any other build of a clone
// with a version it makes up from the commit's time and hash, and that one is a work copy too.
var madeUp = regexp.MustCompile(`\d{14}-[0-9a-f]{12}`)

func versionOf(info *debug.BuildInfo, ok bool) string {
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" || madeUp.MatchString(info.Main.Version) {
		return "dev"
	}

	return info.Main.Version
}

// A start by hand always shows the settings: with its icon tucked away the program would look as
// if it never started. At sign-in it keeps to the tray.
func settingsAtStart(args []string) bool {
	return !slices.Contains(args, tray.Background)
}
