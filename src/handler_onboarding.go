package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// registerOnboardingHandlers wires the onboarding finish: the intent is
// re-validated, everything is persisted in a single config save, autostart
// is applied and the watcher started; the OnboardingFinished fact then swaps
// the window content from the wizard to the main view.
func registerOnboardingHandlers(bus *Bus, s *State, ui *uiRefs, watcher *Watcher) {
	Subscribe(bus, func(e OnboardingFinishRequested) {
		root, err := resolveH3Root(e.Dir)
		if err != nil {
			fyne.Do(func() { dialog.ShowError(err, ui.window) })
			return
		}

		cfg := loadConfig()
		cfg.WatchDir = root
		cfg.Language = string(e.Lang)
		cfg.OnboardingFinished = true
		saveConfig(cfg)

		if e.Autostart && !isStartupEnabled() {
			if err := enableStartup(); err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf(s.T(KeyStartupError), err), ui.window)
				})
			}
		}

		watcher.Start(root)
		bus.Publish(WatchDirChanged{Dir: root})
		bus.Publish(OnboardingFinished{Dir: root})
	})
	Subscribe(bus, func(e OnboardingFinished) {
		fyne.Do(func() {
			relabelMainUI(s, ui)
			ui.dirLabel.SetText(e.Dir)
			ui.window.SetContent(ui.mainContent)
		})
	})
}
