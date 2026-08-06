package main

import (
	"fyne.io/fyne/v2"
)

// registerLanguageHandlers wires language changes: persist the config,
// relabel the widgets and refresh the log list (entries re-render in the new
// language).
func registerLanguageHandlers(bus *Bus, a *App) {
	Subscribe(bus, func(e LanguageChanged) {
		a.mu.Lock()
		a.lang = e.Lang
		watchDir := a.watchDir
		a.mu.Unlock()

		cfg := loadConfig()
		cfg.Language = string(e.Lang)
		saveConfig(cfg)

		fyne.Do(func() {
			a.selectedLabel.SetText(a.T(KeySelectedFolder))
			a.browseBtn.SetText(a.T(KeyChooseDirectory))
			// No dir set — the label shows the placeholder in the previous
			// language; retranslate it.
			if watchDir == "" {
				a.dirLabel.SetText(a.T(KeyNoDirectorySelected))
			}
			a.refreshStartupBtn()
			if a.logList != nil {
				a.logList.Refresh()
			}
		})
	})
}
