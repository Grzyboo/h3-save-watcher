package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

// saveDetectedEvent maps a debounced save-file write to its domain event.
func saveDetectedEvent(path, fileType string) Event {
	switch fileType {
	case "BATTLE":
		return BattleSaveDetected{Path: path}
	case "BATTLE_NC":
		return BattleNonContinuableSaveDetected{Path: path}
	case "TURN_BEGIN":
		return TurnBeginSaveDetected{Path: path}
	case "GAME_BEGIN":
		return GameBeginSaveDetected{Path: path}
	case "TURN_END":
		return TurnEndSaveDetected{Path: path, Turn: turnNumber(path)}
	}
	return nil
}

// turnNumber extracts the turn number from a save filename like "12.GM2".
func turnNumber(path string) int {
	base := filepath.Base(path)
	n, _ := strconv.Atoi(strings.TrimSuffix(base, ".GM2"))
	return n
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
	a.gamesDirWatched = false
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

	// Always watch the root dir: its events let us detect the Games folder
	// being created (or deleted and re-created) while the app is running.
	if err := watcher.Add(dir); err != nil {
		a.addLog(false, KeyLogWatchError, err)
		watcher.Close()
		return
	}

	gamesDir := filepath.Join(dir, "Games")
	gamesDirWatched := false
	if _, err := os.Stat(gamesDir); err == nil {
		if err := watcher.Add(gamesDir); err != nil {
			a.addLog(false, KeyLogWatchError, err)
		} else {
			gamesDirWatched = true
		}
	}

	a.mu.Lock()
	a.watcher = watcher
	a.gamesDirWatched = gamesDirWatched
	a.mu.Unlock()

	absToType := make(map[string]string, len(watchedFiles))
	for _, wf := range watchedFiles {
		abs, _ := filepath.Abs(filepath.Join(dir, wf.relPath))
		absToType[abs] = wf.fileType
	}

	absPasswords, _ := filepath.Abs(filepath.Join(dir, "Games", "passwords.txt"))
	absGames, _ := filepath.Abs(gamesDir)

	// Initial passwords.txt load (handled like a create event), then the
	// watcher-started fact; the bus processes them in publish order.
	a.bus.Publish(PasswordsCreated{Path: absPasswords})
	a.bus.Publish(WatchStarted{Dir: dir})

	go func() {
		type pending struct {
			path     string
			fileType string
			timer    *time.Timer
		}
		debounce := make(map[string]*pending)
		var passwordsTimer *time.Timer
		passwordsCreated := false
		var dmu sync.Mutex

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				absEvent, _ := filepath.Abs(event.Name)

				if absEvent == absGames {
					// The Games folder itself was created (or removed) while
					// watching the root dir — pick it up without a restart.
					if event.Op&fsnotify.Create != 0 {
						a.bus.Publish(GamesDirAppeared{Dir: gamesDir})
					} else if event.Op&fsnotify.Remove != 0 {
						a.bus.Publish(GamesDirRemoved{})
					}
					continue
				}

				if absEvent == absPasswords && (event.Op&(fsnotify.Write|fsnotify.Create)) != 0 {
					created := event.Op&fsnotify.Create != 0
					dmu.Lock()
					passwordsCreated = passwordsCreated || created
					if passwordsTimer != nil {
						passwordsTimer.Reset(watchedFilesDebounce)
					} else {
						passwordsTimer = time.AfterFunc(watchedFilesDebounce, func() {
							dmu.Lock()
							passwordsTimer = nil
							created := passwordsCreated
							passwordsCreated = false
							dmu.Unlock()
							if created {
								a.bus.Publish(PasswordsCreated{Path: absPasswords})
							} else {
								a.bus.Publish(PasswordsModified{Path: absPasswords})
							}
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
							a.bus.Publish(saveDetectedEvent(p.path, p.fileType))
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
								a.bus.Publish(saveDetectedEvent(p.path, p.fileType))
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
