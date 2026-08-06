package main

// registerLogHandlers subscribes the log projection to fact events. For now
// it only covers the watcher-start status; the remaining addLog call sites
// move here in Step 6.
//
// WatchStarted is always published right after PasswordsCreated, so by the
// time this handler runs the gameInfo reflects the freshly parsed file.
func registerLogHandlers(bus *Bus, a *App) {
	Subscribe(bus, func(e WatchStarted) {
		a.mu.Lock()
		hasGameInfo := a.gameInfo.OpponentName != ""
		a.mu.Unlock()
		if hasGameInfo {
			a.addLog(true, KeyLogWatching)
		} else {
			a.addLog(true, KeyLogWatchingWaiting)
		}
	})
}
