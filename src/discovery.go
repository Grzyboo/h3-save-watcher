package main

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

func looksLikeH3(name string) bool {
	low := strings.ToLower(name)
	has3 := strings.Contains(low, "3") || strings.Contains(low, "iii")

	if strings.Contains(low, "hota") {
		return true
	}
	if strings.Contains(low, "heros") {
		return true
	}
	if strings.Contains(low, "heroes of might and magic") && has3 {
		return true
	}
	if strings.Contains(low, "homm") && has3 {
		return true
	}
	if strings.Contains(low, "heroes") && has3 {
		return true
	}
	if strings.Contains(low, "h3") {
		return true
	}
	return false
}

// hotaFilesModTime returns the newest modification time among the HotA
// signature files in the given root dir. Returns zero time if none found.
func hotaFilesModTime(root string) time.Time {
	entries, err := os.ReadDir(root)
	if err != nil {
		return time.Time{}
	}
	var newest time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		isHota := n == "h3hota.exe" || n == "h3hota HD.exe" || n == "HD_Launcher.exe" ||
			strings.HasPrefix(n, "HotA_")
		if !isHota {
			continue
		}
		if info, err := e.Info(); err == nil {
			if t := info.ModTime(); t.After(newest) {
				newest = t
			}
		}
	}
	return newest
}

type searchRoot struct {
	path      string
	recursive bool
}

func platformSearchRoots() []searchRoot {
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "windows":
		var roots []searchRoot
		for _, letter := range "CDEFGHIJKLMNOPQRSTUVWXYZ" {
			drive := string(letter) + `:\`
			if _, err := os.Stat(drive); err == nil {
				roots = append(roots, searchRoot{path: drive, recursive: false})
				gogGames := filepath.Join(drive, "GOG Games")
				if info, err := os.Stat(gogGames); err == nil && info.IsDir() {
					roots = append(roots, searchRoot{path: gogGames, recursive: false})
				}
			}
		}
		return roots

	case "darwin":
		return []searchRoot{
			{path: filepath.Join(home, "Games", "Heroic"), recursive: true},
			{
				path: filepath.Join(home, "Applications",
					"Heroes of Might and Magic III.app",
					"Contents", "SharedSupport", "prefix",
					"drive_c", "GOG Games"),
				recursive: true,
			},
		}

	default: // linux
		return []searchRoot{
			{path: filepath.Join(home, "Games"), recursive: true},
		}
	}
}

// findAllInstallations searches all platform locations and returns every
// valid H3 root, sorted by HotA signature file modification time (newest
// first).
func findAllInstallations() []string {
	var all []string
	for _, sr := range platformSearchRoots() {
		if sr.recursive {
			all = append(all, collectRecursive(sr.path)...)
		} else {
			all = append(all, collectShallow(sr.path)...)
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		return hotaFilesModTime(all[i]).After(hotaFilesModTime(all[j]))
	})
	return all
}

func collectShallow(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var found []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if looksLikeH3(e.Name()) {
			candidate := filepath.Join(dir, e.Name())
			if root, err := resolveH3Root(candidate); err == nil {
				found = append(found, root)
			}
		}
	}
	return found
}

func collectRecursive(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var found []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		sub := filepath.Join(dir, e.Name())
		if looksLikeH3(e.Name()) {
			if root, err := resolveH3Root(sub); err == nil {
				found = append(found, root)
				continue // no need to recurse into a confirmed root
			}
		}
		found = append(found, collectRecursive(sub)...)
	}
	return found
}
