package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// registerGameFolderHandlers wires the game-folder pipeline: the Games dir
// appearing/disappearing at runtime, and game-folder resolution (5s debounce
// after passwords load, 60s retry loop, watch switch + initial scan on
// success).
func registerGameFolderHandlers(bus *Bus, a *App) {
	Subscribe(bus, func(e GamesDirAppeared) { a.onGamesDirAppeared(bus, e.Dir) })
	Subscribe(bus, func(e GamesDirRemoved) { a.onGamesDirRemoved() })
	Subscribe(bus, func(e PasswordsLoaded) { a.scheduleGameFolderWatch(bus) })
	Subscribe(bus, func(e GameFolderResolveRequested) { a.resolveAndWatchGameFolder(bus) })
	Subscribe(bus, func(e GameFolderResolved) { a.switchGameFolder(bus, e.Folder) })
	Subscribe(bus, func(e GameFolderNotFound) { log.Printf("game folder not found: %v", e.Err) })
}

// onGamesDirAppeared adds <root>/Games to the fsnotify watcher after the
// folder appears (it may not have existed when the watcher was started). If
// passwords.txt is already there, a load is triggered right away; otherwise
// its Create event will trigger a load via the regular event branch.
func (a *App) onGamesDirAppeared(bus *Bus, gamesDir string) {
	a.mu.Lock()
	already := a.gamesDirWatched
	watcher := a.watcher
	a.mu.Unlock()
	if already || watcher == nil {
		return
	}

	if err := watcher.Add(gamesDir); err != nil {
		bus.Publish(WatchFailed{Dir: gamesDir, Err: err, Kind: WatchAddFailed})
		return
	}

	a.mu.Lock()
	a.gamesDirWatched = true
	a.mu.Unlock()
	log.Printf("Games folder detected, watching: %s", gamesDir)

	if _, err := os.Stat(filepath.Join(gamesDir, "passwords.txt")); err == nil {
		bus.Publish(PasswordsCreated{Path: filepath.Join(gamesDir, "passwords.txt")})
	}
}

// onGamesDirRemoved drops the watch on <root>/Games after the folder was
// removed, so that a later re-creation triggers GamesDirAppeared again.
func (a *App) onGamesDirRemoved() {
	a.mu.Lock()
	a.gamesDirWatched = false
	watcher := a.watcher
	dir := a.watchDir
	a.mu.Unlock()
	if watcher == nil || dir == "" {
		return
	}
	gamesDir := filepath.Join(dir, "Games")
	_ = watcher.Remove(gamesDir) // the watch may already be gone — ignore
	log.Printf("Games folder removed, will re-watch on re-creation: %s", gamesDir)
}

// scheduleGameFolderWatch schedules a game folder resolution attempt after
// a 5-second debounce. It cancels any previously scheduled attempt or retry loop.
func (a *App) scheduleGameFolderWatch(bus *Bus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.gameFolderCancel != nil {
		a.gameFolderCancel()
		a.gameFolderCancel = nil
	}
	if a.gameFolderDebounce != nil {
		a.gameFolderDebounce.Reset(gameFolderDebounceDelay)
	} else {
		a.gameFolderDebounce = time.AfterFunc(gameFolderDebounceDelay, func() {
			a.mu.Lock()
			a.gameFolderDebounce = nil
			a.mu.Unlock()
			bus.Publish(GameFolderResolveRequested{})
		})
	}
}

// resolveAndWatchGameFolder determines the current game folder and publishes
// GameFolderResolved. If the game folder cannot be found, it publishes
// GameFolderNotFound and retries every minute.
func (a *App) resolveAndWatchGameFolder(bus *Bus) {
	a.mu.Lock()
	dir := a.watchDir
	info := a.gameInfo
	a.mu.Unlock()

	if dir == "" || info.OpponentName == "" {
		return
	}

	folder, err := determineGameFolder(dir, info)
	if err != nil {
		bus.Publish(GameFolderNotFound{Opponent: info.OpponentName, Err: err})

		ctx, cancel := context.WithCancel(context.Background())
		a.mu.Lock()
		if a.gameFolderCancel != nil {
			a.gameFolderCancel()
		}
		a.gameFolderCancel = cancel
		a.mu.Unlock()

		go func() {
			ticker := time.NewTicker(gameFolderRetryInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					bus.Publish(GameFolderResolveRequested{})
				}
			}
		}()
		return
	}

	a.mu.Lock()
	if a.gameFolderCancel != nil {
		a.gameFolderCancel()
		a.gameFolderCancel = nil
	}
	a.mu.Unlock()

	bus.Publish(GameFolderResolved{Folder: folder})
}

// switchGameFolder removes the old game folder watch and starts watching the
// new one, then runs the initial scan.
func (a *App) switchGameFolder(bus *Bus, folder string) {
	a.mu.Lock()
	oldFolder := a.watchedGameFolder
	watcher := a.watcher
	a.watchedGameFolder = folder
	a.mu.Unlock()

	if oldFolder != "" && oldFolder != folder && watcher != nil {
		_ = watcher.Remove(oldFolder)
	}

	if watcher != nil {
		if err := watcher.Add(folder); err != nil {
			bus.Publish(WatchFailed{Dir: folder, Err: err, Kind: WatchAddFailed})
			return
		}
	}

	log.Printf("watching game folder: %s", folder)
	a.uploadExistingGameFolderFiles(folder)
}

// uploadExistingGameFolderFiles scans the game folder and publishes the
// save-detected events for files that were not sent yet, so all uploads go
// through one code path.
func (a *App) uploadExistingGameFolderFiles(folder string) {
	// Unset initial run to send all the future files normally
	defer func() { a.isInitialRun = false }()

	entries, err := os.ReadDir(folder)
	if err != nil {
		// Not an upload attempt (no matching UploadStarted); the log
		// projection still maps this to the read-error entry.
		a.bus.Publish(UploadFailed{Path: folder, Kind: UploadErrRead, Err: err})
		return
	}

	absFolder, _ := filepath.Abs(folder)

	var initialFiles []string
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		fname := entry.Name()
		var ft string
		switch {
		case endTurnFile.MatchString(fname):
			ft = "TURN_END"
		case fname == "GAME_BEGIN.GM2":
			ft = "GAME_BEGIN"
		}
		if ft == "" {
			continue
		}

		a.sentFoldersMu.Lock()
		alreadySent := a.sentFoldersCache.hasFile(absFolder, fname)
		if !alreadySent {
			a.sentFoldersCache.addFile(absFolder, fname)
			if a.isInitialRun {
				initialFiles = append(initialFiles, fname)
			}
		}
		a.sentFoldersMu.Unlock()

		if alreadySent {
			log.Printf("skipping already sent file: %s", fname)
			continue
		}
		if a.isInitialRun {
			log.Printf("initial run: marking file as sent without uploading: %s", fname)
			continue
		}
		a.bus.Publish(saveDetectedEvent(filepath.Join(folder, fname), ft))
	}

	if a.isInitialRun && len(initialFiles) > 0 {
		a.sentFoldersMu.Lock()
		a.sentFoldersCache.setInfo(absFolder, fmt.Sprintf("Initial run, didn't send files: %s", strings.Join(initialFiles, ", ")))
		if err := a.sentFoldersCache.save(); err != nil {
			log.Printf("failed to save sent folders cache: %v", err)
		}
		a.sentFoldersMu.Unlock()
	}
}

// determineGameFolder finds the best-matching game folder under
// <root>/Games/HotA Random/<opponent-name>/ by comparing folder name timestamps
// against the reference time from passwords.txt.
func determineGameFolder(root string, info GameInfo) (string, error) {
	opponentDir := filepath.Join(root, "Games", "HotA Random", info.OpponentName)
	entries, err := os.ReadDir(opponentDir)
	if err != nil {
		return "", err
	}

	refTime := info.GameTime.Add(-1 * time.Minute)
	folderTimeRe := regexp.MustCompile(`^(\d{4})\.(\d{2})\.(\d{2}) (\d{2});(\d{2}) `)

	type match struct {
		path string
		mod  time.Time
	}
	var candidates []match

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		m := folderTimeRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		parsed, err := time.Parse("2006.01.02 15;04", m[1]+"."+m[2]+"."+m[3]+" "+m[4]+";"+m[5])
		if err != nil {
			continue
		}
		if parsed.Before(refTime) {
			continue
		}
		fullPath := filepath.Join(opponentDir, name)
		fi, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		candidates = append(candidates, match{path: fullPath, mod: fi.ModTime()})
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no matching game folder found in %s", opponentDir)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].mod.After(candidates[j].mod)
	})

	return candidates[0].path, nil
}
