//go:build !windows

package main

import "errors"

var errWindowsOnly = errors.New("normprichel: works on Windows only")

func run() error {
	return errWindowsOnly
}
