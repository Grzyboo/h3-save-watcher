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

	"github.com/fsnotify/fsnotify"
)

var endTurnFile = regexp.MustCompile(`^\d+\.GM2$`)

const (
	watchedFilesDebounce    = 400 * time.Millisecond
	gameFolderDebounceDelay = 5 * time.Second
	gameFolderRetryInterval = 60 * time.Second
)

var watchedFiles = []struct {
	relPath  string
	fileType string
}{
	{"Games/BATTLE.GM2", "BATTLE"},
	{"Games/BATTLE_non_continuable.GM2", "BATTLE_NC"},
	{"Games/TURN_BEGIN.GM2", "TURN_BEGIN"},
}

func (a *App) startWatcher(dir string) {
	a.mu.Lock()
	if a.watcher != nil {
		_ = a.watcher.Close()
		a.watcher = nil
	}
	if a.gameFolderCancel != nil {
		a.gameFolderCancel()
		a.gameFolderCancel = nil
	}
	if a.gameFolderDebounce != nil {
		a.gameFolderDebounce.Stop()
		a.gameFolderDebounce = nil
	}
	a.watchedGameFolder = ""
	a.watchDir = dir
	a.mu.Unlock()

	if dir == "" {
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		a.addLog(false, KeyLogWatcherInitError, err)
		return
	}

	gamesDir := filepath.Join(dir, "Games")
	watchTarget := gamesDir
	if _, err := os.Stat(gamesDir); os.IsNotExist(err) {
		watchTarget = dir
	}

	if err := watcher.Add(watchTarget); err != nil {
		a.addLog(false, KeyLogWatchError, err)
		watcher.Close()
		return
	}

	a.mu.Lock()
	a.watcher = watcher
	a.mu.Unlock()

	absToType := make(map[string]string, len(watchedFiles))
	for _, wf := range watchedFiles {
		abs, _ := filepath.Abs(filepath.Join(dir, wf.relPath))
		absToType[abs] = wf.fileType
	}

	a.loadPasswordsFile(dir)

	a.addLog(true, KeyLogWatching)

	absPasswords, _ := filepath.Abs(filepath.Join(dir, "Games", "passwords.txt"))

	go func() {
		type pending struct {
			path     string
			fileType string
			timer    *time.Timer
		}
		debounce := make(map[string]*pending)
		var passwordsTimer *time.Timer
		var dmu sync.Mutex

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				absEvent, _ := filepath.Abs(event.Name)

				if absEvent == absPasswords && (event.Op&(fsnotify.Write|fsnotify.Create)) != 0 {
					dmu.Lock()
					if passwordsTimer != nil {
						passwordsTimer.Reset(watchedFilesDebounce)
					} else {
						passwordsTimer = time.AfterFunc(watchedFilesDebounce, func() {
							dmu.Lock()
							passwordsTimer = nil
							dmu.Unlock()
							a.loadPasswordsFile(dir)
						})
					}
					dmu.Unlock()
					continue
				}

				if fileType, matched := absToType[absEvent]; matched && (event.Op&(fsnotify.Write|fsnotify.Create)) != 0 {
					dmu.Lock()
					if p, exists := debounce[absEvent]; exists {
						p.timer.Reset(watchedFilesDebounce)
					} else {
						p := &pending{path: event.Name, fileType: fileType}
						p.timer = time.AfterFunc(watchedFilesDebounce, func() {
							dmu.Lock()
							delete(debounce, absEvent)
							dmu.Unlock()
							a.uploadFile(p.path, p.fileType)
						})
						debounce[absEvent] = p
					}
					dmu.Unlock()
					continue
				}

				a.mu.Lock()
				gf := a.watchedGameFolder
				a.mu.Unlock()
				if gf != "" && strings.HasPrefix(absEvent, gf+string(os.PathSeparator)) && (event.Op&(fsnotify.Write|fsnotify.Create)) != 0 {
					fname := filepath.Base(event.Name)
					var ft string
					switch {
					case endTurnFile.MatchString(fname) && (event.Op&(fsnotify.Write|fsnotify.Create)) != 0:
						ft = "TURN_END"
					case fname == "GAME_BEGIN.GM2":
						ft = "GAME_BEGIN"
					}
					if ft != "" {
						dmu.Lock()
						if p, exists := debounce[absEvent]; exists {
							p.timer.Reset(watchedFilesDebounce)
						} else {
							p := &pending{path: event.Name, fileType: ft}
							p.timer = time.AfterFunc(watchedFilesDebounce, func() {
								dmu.Lock()
								delete(debounce, absEvent)
								dmu.Unlock()
								a.uploadFile(p.path, p.fileType)
							})
							debounce[absEvent] = p
						}
						dmu.Unlock()
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				a.addLog(false, KeyLogWatcherError, err)
			}
		}
	}()
}

// scheduleGameFolderWatch schedules a game folder resolution attempt after
// a 5-second debounce. It cancels any previously scheduled attempt or retry loop.
func (a *App) scheduleGameFolderWatch() {
	a.mu.Lock()
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
			a.resolveAndWatchGameFolder()
		})
	}
	a.mu.Unlock()
}

// resolveAndWatchGameFolder determines the current game folder and adds it
// to the fsnotify watcher. If the game folder cannot be found, it retries
// every minute.
func (a *App) resolveAndWatchGameFolder() {
	a.mu.Lock()
	dir := a.watchDir
	info := a.gameInfo
	a.mu.Unlock()

	if dir == "" || info.OpponentName == "" {
		return
	}

	folder, err := a.determineGameFolder(dir, info)
	if err != nil {
		log.Printf("game folder not found: %v", err)

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
					a.mu.Lock()
					dir := a.watchDir
					info := a.gameInfo
					a.mu.Unlock()

					folder, err := a.determineGameFolder(dir, info)
					if err != nil {
						log.Printf("game folder retry: %v", err)
						continue
					}

					select {
					case <-ctx.Done():
						return
					default:
					}

					a.switchGameFolder(folder)
					cancel()
					return
				}
			}
		}()
		return
	}

	a.switchGameFolder(folder)
}

// switchGameFolder removes the old game folder watch and starts watching the new one.
func (a *App) switchGameFolder(folder string) {
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
			a.addLog(false, KeyLogWatchError, err)
			return
		}
	}

	log.Printf("watching game folder: %s", folder)
	a.uploadExistingGameFolderFiles(folder)
}

func (a *App) uploadExistingGameFolderFiles(folder string) {
	entries, err := os.ReadDir(folder)
	if err != nil {
		a.addLog(false, KeyLogReadError, folder, err)
		return
	}

	absFolder, _ := filepath.Abs(folder)

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
		if ft != "" {
			a.sentFoldersMu.Lock()
			alreadySent := a.sentFoldersCache.hasFile(absFolder, fname)
			a.sentFoldersMu.Unlock()
			if alreadySent {
				log.Printf("skipping already sent file: %s", fname)
				continue
			}
			a.uploadFile(filepath.Join(folder, fname), ft)
		}
	}
}

// determineGameFolder finds the best-matching game folder under
// <root>/Games/HotA Random/<opponent-name>/ by comparing folder name timestamps
// against the reference time from passwords.txt.
func (a *App) determineGameFolder(root string, info GameInfo) (string, error) {
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
