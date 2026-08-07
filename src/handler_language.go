package main

import "fyne.io/fyne/v2"

// registerLanguageHandlers wires language changes: persist the config,
// relabel the widgets and refresh the log list (entries re-render in the new
// language).
func registerLanguageHandlers(bus *Bus, s *State, ui *uiRefs, logs *logStore) {
	Subscribe(bus, func(e LanguageChanged) {
		s.setLang(e.Lang)

		cfg := loadConfig()
		cfg.Language = string(e.Lang)
		saveConfig(cfg)

		fyne.Do(func() {
			ui.selectedLabel.SetText(s.T(KeySelectedFolder))
			ui.browseBtn.SetText(s.T(KeyChooseDirectory))
			// Check State rather than comparing the label with translated text
			// from the new language.
			if s.watchDir == "" {
				ui.dirLabel.SetText(s.T(KeyNoDirectorySelected))
			}
			refreshStartupBtn(s, ui)
			logs.refresh()
		})
	})
}
