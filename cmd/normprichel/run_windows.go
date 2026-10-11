package main

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/maksim-miliutin/NormPrichel/internal/app"
	"github.com/maksim-miliutin/NormPrichel/internal/foreground"
	"github.com/maksim-miliutin/NormPrichel/internal/hotkeys"
	"github.com/maksim-miliutin/NormPrichel/internal/live"
	"github.com/maksim-miliutin/NormPrichel/internal/overlay"
	"github.com/maksim-miliutin/NormPrichel/internal/profiles"
	"github.com/maksim-miliutin/NormPrichel/internal/settings"
	"github.com/maksim-miliutin/NormPrichel/internal/tray"
)

func run() error {
	first, err := tray.Claim()
	if err != nil {
		return err
	}
	if !first {
		fmt.Fprintln(os.Stderr, "normprichel: another copy is running; it opens its settings page, and this one ends")
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

	language := tray.SystemLanguage()

	// Windows gives a window, its timer and its hotkeys to the thread that made them, and the
	// overlay's message loop serves every window of the thread it runs on.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	screen := &overlay.Window{}
	watcher := foreground.NewWatcher(foreground.System{}, uint32(os.Getpid()))
	loop := &app.Loop{State: state, Overlay: overlay.NewController(screen, watcher, store.Paint), Games: watcher}
	tick := func() { report(loop.Tick()) }

	address := ""
	acted := reporter(os.Stderr)
	shell, err := tray.New(language, tray.Handlers{
		Hotkey: func(action int) {
			acted(loop.Hotkey(action))
			tick()
		},
		Open: func() { acted(tray.Open(address)) },
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

	address, server, err := serve(settings.Page{
		State:    state,
		Pictures: store,
		Windows:  func() ([]foreground.Choice, error) { return foreground.System{}.Choices(uint32(os.Getpid())) },
		Taken:    func(bindings []hotkeys.Binding) []hotkeys.Binding { return hotkeys.Taken(keys.Others(bindings)) },
		Version:  versionOf(debug.ReadBuildInfo()),
		Language: language,
	})
	if err != nil {
		return err
	}
	defer server.Close()

	report(shell.ShowIcon())
	if pageAtStart(os.Args[1:]) {
		report(tray.Open(address))
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		screen.Stop()
	}()

	return screen.Run(tick)
}

func serve(page settings.Page) (string, *http.Server, error) {
	key, err := settings.NewKey(rand.Reader)
	if err != nil {
		return "", nil, err
	}
	listener, err := settings.Listen()
	if err != nil {
		return "", nil, err
	}

	host := listener.Addr().String()
	server := &http.Server{Handler: settings.Guard(key, host, page.Handler()), ReadHeaderTimeout: 10 * time.Second}
	go server.Serve(listener)

	return "http://" + host + "/" + key + "/", server, nil
}
