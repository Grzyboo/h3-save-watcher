package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// registerWatchDirHandlers wires watch-directory changes: the intent is
// validated, persisted and applied (watcher restart); failures show an error
// dialog.
func registerWatchDirHandlers(bus *Bus, a *App) {
	Subscribe(bus, func(e WatchDirChangeRequested) {
		root, err := resolveH3Root(e.Dir)
		if err != nil {
			bus.Publish(WatchDirInvalid{Dir: e.Dir, Err: err})
			return
		}

		fyne.Do(func() { a.dirLabel.SetText(root) })
		cfg := loadConfig()
		cfg.WatchDir = root
		saveConfig(cfg)
		a.startWatcher(root)
		bus.Publish(WatchDirChanged{Dir: root})

		a.mu.Lock()
		firstRun := a.firstRun
		a.mu.Unlock()
		if firstRun && !isStartupEnabled() {
			a.mu.Lock()
			a.firstRun = false
			a.mu.Unlock()
			bus.Publish(StartupToggleRequested{})
		}
	})
	Subscribe(bus, func(e WatchDirInvalid) {
		fyne.Do(func() { dialog.ShowError(e.Err, a.window) })
	})
}
