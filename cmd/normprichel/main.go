package main

import (
	"fmt"
	"io"
	"os"
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
