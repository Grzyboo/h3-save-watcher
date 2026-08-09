package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// registerWatchDirHandlers wires watch-directory changes: the intent is
// validated, persisted and applied by restarting the watcher; failures show
// an error dialog.
func registerWatchDirHandlers(bus *Bus, s *State, ui *uiRefs, watcher *Watcher) {
	Subscribe(bus, func(e WatchDirChangeRequested) {
		root, err := resolveH3Root(e.Dir)
		if err != nil {
			bus.Publish(WatchDirInvalid{Dir: e.Dir, Err: err})
			return
		}

		fyne.Do(func() {
			ui.dirLabel.SetText(root)
			if ui.settingsDirLabel != nil {
				ui.settingsDirLabel.SetText(root)
			}
		})
		cfg := loadConfig()
		cfg.WatchDir = root
		saveConfig(cfg)
		watcher.Start(root)
		bus.Publish(WatchDirChanged{Dir: root})
	})
	Subscribe(bus, func(e WatchDirInvalid) {
		fyne.Do(func() { dialog.ShowError(e.Err, ui.window) })
	})
}
