package main

import (
	"errors"
	"os"
	"path/filepath"
)

// isH3Root reports whether dir is a HotA installation root. Every real
// installation ships one of these executables; looser heuristics (e.g. the
// "HotA_" file prefix) also match installer downloads like
// "HotA_1.8.0_setup.exe" and so would accept folders such as ~/Downloads.
func isH3Root(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch e.Name() {
		case "h3hota.exe", "h3hota HD.exe", "HD_Launcher.exe":
			return true
		}
	}
	return false
}

// resolveH3Root accepts any directory the user might reasonably pick and
// returns the H3 installation root, or an error if it cannot be determined.
//
// Strategy:
//  1. Walk upward from the selected dir — handles: root, Games/, Games/HotA Random/, Games/HotA Random/<name>
//  2. If not found going up, scan immediate children — handles: parent-of-root (e.g. ~/Gog Games)
func resolveH3Root(selected string) (string, error) {
	abs, err := filepath.Abs(selected)
	if err != nil {
		return "", err
	}

	// Walk upward (including selected dir itself).
	cur := abs
	for {
		if isH3Root(cur) {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break // reached filesystem root
		}
		cur = parent
	}

	// Not found going up — try immediate children (user picked parent dir).
	entries, err := os.ReadDir(abs)
	if err != nil {
		return "", errors.New("Could not find H3 installation in or around the selected directory")
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		candidate := filepath.Join(abs, e.Name())
		if isH3Root(candidate) {
			return candidate, nil
		}
	}

	return "", errors.New("Could not find H3 installation in or around the selected directory.\n\nPlease select the valid installation folder of Heroes of Might and Magic III: Horn of the Abyss")
}
