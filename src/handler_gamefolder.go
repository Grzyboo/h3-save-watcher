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
	"sync"
	"time"
)

// gameFolderScheduler owns the game-folder resolve debounce timer and the
// not-found retry loop. Both fire by publishing events. It has its own mutex
// because timers fire on their own goroutines and stop is also called from
// the updater and main (shutdown) goroutines.
type gameFolderScheduler struct {
	mu          sync.Mutex
	debounce    *time.Timer
	retryCancel context.CancelFunc
}

func newGameFolderScheduler() *gameFolderScheduler {
	return &gameFolderScheduler{}
}

// schedule (re)arms the resolve debounce and cancels any retry loop.
func (s *gameFolderScheduler) schedule(delay time.Duration, fire func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.retryCancel != nil {
		s.retryCancel()
		s.retryCancel = nil
	}
	if s.debounce != nil {
		s.debounce.Reset(delay)
		return
	}
	s.debounce = time.AfterFunc(delay, func() {
		s.mu.Lock()
		s.debounce = nil
		s.mu.Unlock()
		fire()
	})
}

// startRetry launches the retry loop, canceling any previous one.
func (s *gameFolderScheduler) startRetry(interval time.Duration, fire func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.retryCancel != nil {
		s.retryCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.retryCancel = cancel
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fire()
			}
		}
	}()
}

// cancelRetry stops the retry loop, if any.
func (s *gameFolderScheduler) cancelRetry() {
	s.mu.Lock()
	if s.retryCancel != nil {
		s.retryCancel()
		s.retryCancel = nil
	}
	s.mu.Unlock()
}

// stop cancels both the debounce and the retry loop.
func (s *gameFolderScheduler) stop() {
	s.mu.Lock()
	if s.debounce != nil {
		s.debounce.Stop()
		s.debounce = nil
	}
	if s.retryCancel != nil {
		s.retryCancel()
		s.retryCancel = nil
	}
	s.mu.Unlock()
}

// registerGameFolderHandlers wires the game-folder pipeline: the Games dir
// appearing/disappearing at runtime, and game-folder resolution (5s debounce
// after passwords load, 60s retry loop, watch switch + initial scan on
// success).
func registerGameFolderHandlers(bus *Bus, s *State, w *Watcher, sched *gameFolderScheduler) {
	Subscribe(bus, func(e GamesDirAppeared) { onGamesDirAppeared(bus, s, w, e.Dir) })
	Subscribe(bus, func(e GamesDirRemoved) { onGamesDirRemoved(s, w) })
	Subscribe(bus, func(e PasswordsLoaded) {
		sched.schedule(gameFolderDebounceDelay, func() { bus.Publish(GameFolderResolveRequested{}) })
	})
	Subscribe(bus, func(e GameFolderResolveRequested) { resolveAndWatchGameFolder(bus, s, sched) })
	Subscribe(bus, func(e GameFolderResolved) { switchGameFolder(bus, s, w, e.Folder) })
	Subscribe(bus, func(e GameFolderNotFound) { log.Printf("game folder not found: %v", e.Err) })

	// A watcher (re)start resets all watch state. WatchStarted is published
	// right after PasswordsCreated, so gameInfo is already fresh here.
	Subscribe(bus, func(e WatchStarted) {
		s.watchDir = e.Dir
		s.watchedGameFolder = ""
		s.gamesDirWatched = e.GamesDirWatched
		sched.stop()
	})
}

// onGamesDirAppeared adds <root>/Games to the fsnotify watcher after the
// folder appears (it may not have existed when the watcher was started). If
// passwords.txt is already there, a load is triggered right away; otherwise
// its Create event will trigger a load via the regular event branch.
func onGamesDirAppeared(bus *Bus, s *State, w *Watcher, gamesDir string) {
	if s.gamesDirWatched {
		return
	}

	if err := w.AddDir(gamesDir); err != nil {
		bus.Publish(WatchFailed{Dir: gamesDir, Err: err, Kind: WatchAddFailed})
		return
	}

	s.gamesDirWatched = true
	log.Printf("Games folder detected, watching: %s", gamesDir)

	if _, err := os.Stat(filepath.Join(gamesDir, "passwords.txt")); err == nil {
		bus.Publish(PasswordsCreated{Path: filepath.Join(gamesDir, "passwords.txt")})
	}
}

// onGamesDirRemoved drops the watch on <root>/Games after the folder was
// removed, so that a later re-creation triggers GamesDirAppeared again.
func onGamesDirRemoved(s *State, w *Watcher) {
	s.gamesDirWatched = false
	if s.watchDir == "" {
		return
	}
	gamesDir := filepath.Join(s.watchDir, "Games")
	w.RemoveDir(gamesDir)
	log.Printf("Games folder removed, will re-watch on re-creation: %s", gamesDir)
}

// resolveAndWatchGameFolder determines the current game folder and publishes
// GameFolderResolved. If the game folder cannot be found, it publishes
// GameFolderNotFound and retries every minute.
func resolveAndWatchGameFolder(bus *Bus, s *State, sched *gameFolderScheduler) {
	if s.watchDir == "" || s.gameInfo.OpponentName == "" {
		return
	}

	folder, err := determineGameFolder(s.watchDir, s.gameInfo)
	if err != nil {
		bus.Publish(GameFolderNotFound{Opponent: s.gameInfo.OpponentName, Err: err})
		sched.startRetry(gameFolderRetryInterval, func() { bus.Publish(GameFolderResolveRequested{}) })
		return
	}

	sched.cancelRetry()
	bus.Publish(GameFolderResolved{Folder: folder})
}

// switchGameFolder removes the old game folder watch and starts watching the
// new one, then runs the initial scan.
func switchGameFolder(bus *Bus, s *State, w *Watcher, folder string) {
	oldFolder := s.watchedGameFolder
	s.watchedGameFolder = folder
	w.SetGameFolder(folder)

	if oldFolder != "" && oldFolder != folder {
		w.RemoveDir(oldFolder)
	}

	if err := w.AddDir(folder); err != nil {
		bus.Publish(WatchFailed{Dir: folder, Err: err, Kind: WatchAddFailed})
		return
	}

	log.Printf("watching game folder: %s", folder)
	uploadExistingGameFolderFiles(bus, s, folder)
}

// uploadExistingGameFolderFiles scans the game folder and publishes the
// save-detected events for files that were not sent yet, so all uploads go
// through one code path.
func uploadExistingGameFolderFiles(bus *Bus, s *State, folder string) {
	// Unset initial run to send all the future files normally
	defer func() { s.isInitialRun = false }()

	entries, err := os.ReadDir(folder)
	if err != nil {
		// Not an upload attempt (no matching UploadStarted); the log
		// projection still maps this to the read-error entry.
		bus.Publish(UploadFailed{Path: folder, Kind: UploadErrRead, Err: err})
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

		alreadySent := s.sentFoldersCache.hasFile(absFolder, fname)
		if !alreadySent {
			s.sentFoldersCache.addFile(absFolder, fname)
			if s.isInitialRun {
				initialFiles = append(initialFiles, fname)
			}
		}

		if alreadySent {
			log.Printf("skipping already sent file: %s", fname)
			continue
		}
		if s.isInitialRun {
			log.Printf("initial run: marking file as sent without uploading: %s", fname)
			continue
		}
		bus.Publish(saveDetectedEvent(filepath.Join(folder, fname), ft))
	}

	if s.isInitialRun && len(initialFiles) > 0 {
		s.sentFoldersCache.setInfo(absFolder, fmt.Sprintf("Initial run, didn't send files: %s", strings.Join(initialFiles, ", ")))
		if err := s.sentFoldersCache.save(); err != nil {
			log.Printf("failed to save sent folders cache: %v", err)
		}
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
