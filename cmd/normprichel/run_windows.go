package main

import (
	"os"
	"os/signal"

	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/overlay"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
)

func run() error {
	report := reporter(os.Stderr)
	report(overlay.DeclareDPIAwareness())

	dir, err := profiles.DefaultDir()
	if err != nil {
		return err
	}
	store := profiles.Store{Disk: profiles.OS{}, Dir: dir}
	settings, err := store.Load()
	report(err)

	window := &overlay.Window{}
	watcher := foreground.NewWatcher(foreground.System{}, uint32(os.Getpid()))
	controller := overlay.NewController(window, watcher, store.Paint)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		window.Stop()
	}()

	return window.Run(func() { report(controller.Refresh(settings)) })
}
