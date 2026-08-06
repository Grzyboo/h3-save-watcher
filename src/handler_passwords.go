package main

import (
	"os"
)

// registerPasswordsHandlers wires the passwords.txt pipeline:
// PasswordsCreated/Modified → parse → PasswordsLoaded/PasswordsInvalid.
func registerPasswordsHandlers(bus *Bus, a *App) {
	// handle reads and parses passwords.txt at path, updates gameInfo and
	// publishes the outcome. Runs on the bus goroutine.
	handle := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			a.mu.Lock()
			a.gameInfo = GameInfo{}
			a.mu.Unlock()
			a.addLog(false, KeyLogPasswordsReadError, err)
			bus.Publish(PasswordsInvalid{Path: path, Err: err, Kind: PasswordsReadFailed})
			return
		}

		info, err := parsePasswordsTxt(data)
		if err != nil {
			a.mu.Lock()
			a.gameInfo = GameInfo{}
			a.mu.Unlock()
			a.addLog(false, KeyLogPasswordsParseError, err)
			bus.Publish(PasswordsInvalid{Path: path, Err: err, Kind: PasswordsParseFailed})
			return
		}

		// Record the file's modification time as the game reference time - TODO: read from file contents?
		if fi, err := os.Stat(path); err == nil {
			info.GameTime = fi.ModTime()
		}

		a.mu.Lock()
		changed := a.gameInfo.PlayerName != info.PlayerName ||
			a.gameInfo.OpponentName != info.OpponentName ||
			a.gameInfo.Password != info.Password
		a.gameInfo = info
		a.mu.Unlock()

		if changed {
			a.addLog(true, KeyLogPasswordsLoaded, info.PlayerName, info.OpponentName)
		}

		bus.Publish(PasswordsLoaded{Info: info, Changed: changed})
		a.scheduleGameFolderWatch()
	}

	Subscribe(bus, func(e PasswordsCreated) { handle(e.Path) })
	Subscribe(bus, func(e PasswordsModified) { handle(e.Path) })
}
