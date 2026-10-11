package main

import (
	"os"
	"os/signal"
	"runtime"

	"github.com/maksim-miliutin/NormPrichel/internal/app"
	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/live"
	"github.com/maksim-miliutin/NormPrichel/internal/overlay"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
	"github.com/maksim-miliutin/NormPrichel/internal/tray"
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
	state := live.New(settings, store, reporter(os.Stderr))
	defer state.Close()

	// Windows gives a window, its timer and its hotkeys to the thread that made them, and the
	// overlay's message loop serves every window of the thread it runs on.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	screen := &overlay.Window{}
	watcher := foreground.NewWatcher(foreground.System{}, uint32(os.Getpid()))
	loop := &app.Loop{State: state, Overlay: overlay.NewController(screen, watcher, store.Paint), Games: watcher}
	tick := func() { report(loop.Tick()) }

	pressed := reporter(os.Stderr)
	shell, err := tray.New(tray.Handlers{Hotkey: func(action int) {
		pressed(loop.Hotkey(action))
		tick()
	}})
	if err != nil {
		return err
	}
	defer shell.Close()

	keys := hotkeys.NewKeys(hotkeys.System{Window: shell.Handle()})
	defer keys.Hold(nil)
	loop.Keys = keys

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		screen.Stop()
	}()

	return screen.Run(tick)
}
