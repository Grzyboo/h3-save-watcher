package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GameInfo holds data parsed from the last entry of passwords.txt.
type GameInfo struct {
	IsHost       bool
	PlayerName   string
	OpponentName string
	Password     string
}

// badgeChars are the special prefix characters that mark donator/moderator status.
var badgeChars = []byte{0x17, 0x19, 0x1F}

// entryRe matches a passwords.txt line.
// Group 1: host flag (h or j)
// Group 2: player name (with possible badge chars/trailing })
// Group 3: opponent name (with possible badge chars/trailing })
// Group 4: password
var entryRe = regexp.MustCompile(`^\[([hj])\]\t[^\t]+\t[^\t]+\t(.+) \(vs (.+)\) password:\t(\S+)$`)

// stripBadges removes leading badge bytes (0x17, 0x19, 0x1F) and a trailing '}'.
func stripBadges(s string) string {
	b := []byte(s)
	for len(b) > 0 {
		trimmed := false
		for _, ch := range badgeChars {
			if b[0] == ch {
				b = b[1:]
				trimmed = true
				break
			}
		}
		if !trimmed {
			break
		}
	}
	b = bytes.TrimRight(b, "}")
	return string(b)
}

// parsePasswordsTxt parses the content of passwords.txt and returns
// a GameInfo built from the last non-empty entry.
func parsePasswordsTxt(data []byte) (GameInfo, error) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

	// Find the last non-empty line.
	last := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			last = lines[i]
			break
		}
	}
	if last == "" {
		return GameInfo{}, fmt.Errorf("passwords.txt: no entries found")
	}

	m := entryRe.FindStringSubmatch(last)
	if m == nil {
		return GameInfo{}, fmt.Errorf("passwords.txt: could not parse entry: %q", last)
	}

	info := GameInfo{
		IsHost:       m[1] == "h",
		PlayerName:   stripBadges(m[2]),
		OpponentName: stripBadges(m[3]),
		Password:     m[4],
	}
	log.Printf("passwords.txt parsed: host=%v player=%q opponent=%q", info.IsHost, info.PlayerName, info.OpponentName)
	return info, nil
}

// loadPasswordsFile reads Games/passwords.txt from the given H3 root dir,
// parses the last entry, and stores the result in a.gameInfo.
// On any error the existing gameInfo is erased and a log entry is added.
func (a *App) loadPasswordsFile(dir string) {
	path := filepath.Join(dir, "Games", "passwords.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		a.mu.Lock()
		a.gameInfo = GameInfo{}
		a.mu.Unlock()
		a.addLog(false, KeyLogPasswordsReadError, err)
		return
	}

	info, err := parsePasswordsTxt(data)
	if err != nil {
		a.mu.Lock()
		a.gameInfo = GameInfo{}
		a.mu.Unlock()
		a.addLog(false, KeyLogPasswordsParseError, err)
		return
	}

	a.mu.Lock()
	a.gameInfo = info
	a.mu.Unlock()
}
