package main

import (
	"os"
)

// registerPasswordsHandlers wires the passwords.txt pipeline:
// PasswordsCreated/Modified → parse → PasswordsLoaded/PasswordsInvalid.
func registerPasswordsHandlers(bus *Bus, s *State) {
	// handle reads and parses passwords.txt at path, updates gameInfo and
	// publishes the outcome. Runs on the bus goroutine.
	handle := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			s.gameInfo = GameInfo{}
			bus.Publish(PasswordsInvalid{Path: path, Err: err, Kind: PasswordsReadFailed})
			return
		}

		info, err := parsePasswordsTxt(data)
		if err != nil {
			s.gameInfo = GameInfo{}
			bus.Publish(PasswordsInvalid{Path: path, Err: err, Kind: PasswordsParseFailed})
			return
		}

		// Record the file's modification time as the game reference time - TODO: read from file contents?
		if fi, err := os.Stat(path); err == nil {
			info.GameTime = fi.ModTime()
		}

		changed := s.gameInfo.PlayerName != info.PlayerName ||
			s.gameInfo.OpponentName != info.OpponentName ||
			s.gameInfo.Password != info.Password
		s.gameInfo = info

		bus.Publish(PasswordsLoaded{Info: info, Changed: changed})
	}

	Subscribe(bus, func(e PasswordsCreated) { handle(e.Path) })
	Subscribe(bus, func(e PasswordsModified) { handle(e.Path) })
}
