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

// saveDetectedEvent maps a debounced save-file write or backfill candidate to
// its domain event.
func saveDetectedEvent(path, fileType string, backfill bool) Event {
	switch fileType {
	case "BATTLE":
		return BattleSaveDetected{Path: path}
	case "BATTLE_NC":
		return BattleNonContinuableSaveDetected{Path: path}
	case "TURN_BEGIN":
		return TurnBeginSaveDetected{Path: path}
	case "GAME_BEGIN":
		return GameBeginSaveDetected{Path: path, Backfill: backfill}
	case "TURN_END":
		return TurnEndSaveDetected{Path: path, Turn: turnNumber(path), Backfill: backfill}
	}
	return nil
}

// turnNumber extracts the turn number from a save filename like "12.GM2".
func turnNumber(path string) int {
	base := filepath.Base(path)
	n, _ := strconv.Atoi(strings.TrimSuffix(base, ".GM2"))
	return n
}

// Watcher is the fsnotify producer: it owns the fsnotify watcher, debounces
// file events and publishes domain events on the bus. It contains no business
// logic. All methods are safe to call from any goroutine.
type Watcher struct {
	bus *Bus

	mu         sync.Mutex
	fsn        *fsnotify.Watcher
	gameFolder string // current game folder, used by the read loop to route save events
}

func NewWatcher(bus *Bus) *Watcher {
	return &Watcher{bus: bus}
}

// Start (re)starts the watcher on the given H3 root dir: closes any previous
// watcher, sets up fsnotify, publishes the initial PasswordsCreated and
// WatchStarted, and launches the event read loop.
func (w *Watcher) Start(dir string) {
	w.mu.Lock()
	if w.fsn != nil {
		_ = w.fsn.Close()
		w.fsn = nil
	}
	w.gameFolder = ""
	w.mu.Unlock()

	if dir == "" {
		return
	}

	fsn, err := fsnotify.NewWatcher()
	if err != nil {
		w.bus.Publish(WatchFailed{Dir: dir, Err: err, Kind: WatchInitFailed})
		return
	}

	// Always watch the root dir: its events let us detect the Games folder
	// being created (or deleted and re-created) while the app is running.
	if err := fsn.Add(dir); err != nil {
		w.bus.Publish(WatchFailed{Dir: dir, Err: err, Kind: WatchAddFailed})
		fsn.Close()
		return
	}

	gamesDir := filepath.Join(dir, "Games")
	gamesDirWatched := false
	if _, err := os.Stat(gamesDir); err == nil {
		if err := fsn.Add(gamesDir); err != nil {
			w.bus.Publish(WatchFailed{Dir: gamesDir, Err: err, Kind: WatchAddFailed})
		} else {
			gamesDirWatched = true
		}
	}

	w.mu.Lock()
	w.fsn = fsn
	w.mu.Unlock()

	absToType := make(map[string]string, len(watchedFiles))
	for _, wf := range watchedFiles {
		abs, _ := filepath.Abs(filepath.Join(dir, wf.relPath))
		absToType[abs] = wf.fileType
	}

	absPasswords, _ := filepath.Abs(filepath.Join(dir, "Games", "passwords.txt"))
	absGames, _ := filepath.Abs(gamesDir)

	// Initial passwords.txt load (handled like a create event), then the
	// watcher-started fact; the bus processes them in publish order.
	w.bus.Publish(PasswordsCreated{Path: absPasswords})
	w.bus.Publish(WatchStarted{Dir: dir, GamesDirWatched: gamesDirWatched})

	go w.readLoop(fsn, dir, gamesDir, absGames, absPasswords, absToType)
}

// AddDir adds a directory to the current watch (the Games dir or a game
// folder). It is a no-op when the watcher is not running (app shutdown).
func (w *Watcher) AddDir(dir string) error {
	w.mu.Lock()
	fsn := w.fsn
	w.mu.Unlock()
	if fsn == nil {
		return nil
	}
	return fsn.Add(dir)
}

// RemoveDir drops a directory from the current watch; errors are ignored
// (the watch may already be gone).
func (w *Watcher) RemoveDir(dir string) {
	w.mu.Lock()
	fsn := w.fsn
	w.mu.Unlock()
	if fsn != nil {
		_ = fsn.Remove(dir)
	}
}

// SetGameFolder records the current game folder for save-event routing.
func (w *Watcher) SetGameFolder(folder string) {
	w.mu.Lock()
	w.gameFolder = folder
	w.mu.Unlock()
}

// Close stops the watcher, if running.
func (w *Watcher) Close() error {
	w.mu.Lock()
	fsn := w.fsn
	w.fsn = nil
	w.gameFolder = ""
	w.mu.Unlock()
	if fsn == nil {
		return nil
	}
	return fsn.Close()
}

// readLoop is the fsnotify event pump: it debounces raw events and publishes
// the corresponding domain events. It runs on its own goroutine and never
// touches State.
func (w *Watcher) readLoop(fsn *fsnotify.Watcher, dir, gamesDir, absGames, absPasswords string, absToType map[string]string) {
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
		case event, ok := <-fsn.Events:
			if !ok {
				return
			}
			absEvent, _ := filepath.Abs(event.Name)

			if absEvent == absGames {
				// The Games folder itself was created (or removed) while
				// watching the root dir — pick it up without a restart.
				if event.Op&fsnotify.Create != 0 {
					w.bus.Publish(GamesDirAppeared{Dir: gamesDir})
				} else if event.Op&fsnotify.Remove != 0 {
					w.bus.Publish(GamesDirRemoved{})
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
							w.bus.Publish(PasswordsCreated{Path: absPasswords})
						} else {
							w.bus.Publish(PasswordsModified{Path: absPasswords})
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
						w.bus.Publish(saveDetectedEvent(p.path, p.fileType, false))
					})
					debounce[absEvent] = p
				}
				dmu.Unlock()
				continue
			}

			w.mu.Lock()
			gf := w.gameFolder
			w.mu.Unlock()
			if gf != "" && strings.HasPrefix(absEvent, gf+string(os.PathSeparator)) && (event.Op&(fsnotify.Write|fsnotify.Create)) != 0 {
				fname := filepath.Base(event.Name)
				var ft string
				switch {
				case endTurnFile.MatchString(fname):
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
							w.bus.Publish(saveDetectedEvent(p.path, p.fileType, false))
						})
						debounce[absEvent] = p
					}
					dmu.Unlock()
				}
			}

		case err, ok := <-fsn.Errors:
			if !ok {
				return
			}
			w.bus.Publish(WatchFailed{Dir: dir, Err: err, Kind: WatchRuntimeError})
		}
	}
}
