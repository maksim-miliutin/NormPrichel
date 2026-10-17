package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"

	"github.com/maksim-miliutin/NormPrichel/internal/app"
	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/live"
	"github.com/maksim-miliutin/NormPrichel/internal/mouse"
	"github.com/maksim-miliutin/NormPrichel/internal/overlay"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
	"github.com/maksim-miliutin/NormPrichel/internal/settings"
	"github.com/maksim-miliutin/NormPrichel/internal/tray"
	"github.com/maksim-miliutin/NormPrichel/internal/window"
)

func run() error {
	first, err := tray.Claim()
	if err != nil {
		return err
	}
	if !first {
		fmt.Fprintln(os.Stderr, "normprichel: another copy is running; it shows its settings, and this one ends")
		return tray.Wake()
	}

	report := reporter(os.Stderr)
	report(overlay.DeclareDPIAwareness())

	dir, err := profiles.DefaultDir()
	if err != nil {
		return err
	}
	store := profiles.Store{Disk: profiles.OS{}, Dir: dir}
	loaded, err := store.Load()
	report(err)
	state := live.New(loaded, store, reporter(os.Stderr))
	defer state.Close()

	windowsLanguage := tray.SystemLanguage()
	speaks := func() string { return settings.Language(state.Current().Language, windowsLanguage) }

	// Windows gives a window, its timer and its hotkeys to the thread that made them, and the
	// overlay's message loop serves every window of the thread it runs on.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	screen := &overlay.Window{}
	watcher := foreground.NewWatcher(foreground.System{}, uint32(os.Getpid()))
	loop := &app.Loop{State: state, Overlay: overlay.NewController(screen, watcher, store.Paint), Games: watcher, Mouse: mouse.System{}}
	screen.Often = loop.Watching
	tick := func() { report(loop.Tick()) }

	var panel *window.Window
	acted := reporter(os.Stderr)
	shell, err := tray.New(speaks, tray.Handlers{
		Hotkey: func(action int) {
			acted(loop.Hotkey(action))
			tick()
		},
		Open: func() { panel.Show() },
		Toggle: func() {
			acted(loop.Toggle())
			tick()
		},
		Exit: screen.Stop,
		Fail: acted,
	})
	if err != nil {
		return err
	}
	defer shell.Close()

	keys := hotkeys.NewKeys(hotkeys.System{Window: shell.Handle()})
	defer keys.Hold(nil)
	loop.Keys = keys

	panel, err = window.New(&settings.Form{
		State:           state,
		Pictures:        store,
		Windows:         func() ([]foreground.Choice, error) { return foreground.System{}.Choices(uint32(os.Getpid())) },
		Taken:           func(bindings []hotkeys.Binding) []hotkeys.Binding { return hotkeys.Taken(keys.Others(bindings)) },
		Autostart:       tray.Autostart,
		SwitchAutostart: tray.SetAutostart,
		Version:         versionOf(debug.ReadBuildInfo()),
		WindowsLanguage: windowsLanguage,
	}, tick, acted)
	if err != nil {
		return err
	}
	defer panel.Close()
	report(panel.SetIcon(tray.NewIcon))
	screen.Dialog = panel.Handle

	report(shell.ShowIcon())
	if settingsAtStart(os.Args[1:]) {
		panel.Show()
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		screen.Stop()
	}()

	return screen.Run(tick)
}
