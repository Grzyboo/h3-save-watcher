package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

var endTurnFile = regexp.MustCompile(`^\d+\.GM2$`)

var watchedFiles = []struct {
	relPath  string
	fileType string
}{
	{"Games/BATTLE.GM2", "BATTLE"},
	{"Games/BATTLE_non_continuable.GM2", "BATTLE_NC"},
}

func (a *App) startWatcher(dir string) {
	a.mu.Lock()
	if a.watcher != nil {
		_ = a.watcher.Close()
		a.watcher = nil
	}
	// Stop any running hota poller before starting a new one.
	if a.stopPoll != nil {
		close(a.stopPoll)
		a.stopPoll = nil
	}
	a.watchDir = dir
	a.mu.Unlock()

	if dir == "" {
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		a.addLog(false, fmt.Sprintf(a.T(KeyLogWatcherInitError), err))
		return
	}

	// Watch Games/ subdir if it exists; otherwise fall back to root dir.
	gamesDir := filepath.Join(dir, "Games")
	watchTarget := gamesDir
	if _, err := os.Stat(gamesDir); os.IsNotExist(err) {
		watchTarget = dir
	}

	if err := watcher.Add(watchTarget); err != nil {
		a.addLog(false, fmt.Sprintf(a.T(KeyLogWatchError), err))
		watcher.Close()
		return
	}

	a.mu.Lock()
	a.watcher = watcher
	a.mu.Unlock()

	// Build abs path -> type map and log each watched file.
	absToType := make(map[string]string, len(watchedFiles))
	for _, wf := range watchedFiles {
		abs, _ := filepath.Abs(filepath.Join(dir, wf.relPath))
		absToType[abs] = wf.fileType
	}

	a.addLog(true, a.T(KeyLogWatching))

	// Start the HotA Random poller.
	stopPoll := make(chan struct{})
	a.mu.Lock()
	a.stopPoll = stopPoll
	a.mu.Unlock()
	go a.startHotaPoller(dir, stopPoll)

	go func() {
		type pending struct {
			path     string
			fileType string
			timer    *time.Timer
		}
		debounce := make(map[string]*pending)
		var dmu sync.Mutex

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				absEvent, _ := filepath.Abs(event.Name)
				if fileType, matched := absToType[absEvent]; matched && (event.Op&(fsnotify.Write|fsnotify.Create)) != 0 {
					dmu.Lock()
					if p, exists := debounce[absEvent]; exists {
						p.timer.Reset(time.Second)
					} else {
						p := &pending{path: event.Name, fileType: fileType}
						p.timer = time.AfterFunc(time.Second, func() {
							dmu.Lock()
							delete(debounce, absEvent)
							dmu.Unlock()
							a.uploadFile(p.path, p.fileType)
						})
						debounce[absEvent] = p
					}
					dmu.Unlock()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				a.addLog(false, fmt.Sprintf(a.T(KeyLogWatcherError), err))
			}
		}
	}()
}

// snapshotHotaFiles walks hotaDir and returns a set of relative paths for all
// existing XXX.GM2 files. Used to establish the baseline on startup — these
// files are never uploaded.
func snapshotHotaFiles(hotaDir string) map[string]struct{} {
	known := make(map[string]struct{})
	_ = filepath.WalkDir(hotaDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if endTurnFile.MatchString(d.Name()) {
			rel, _ := filepath.Rel(hotaDir, path)
			known[rel] = struct{}{}
		}
		return nil
	})
	log.Printf("hota snapshot: %d existing files under %s", len(known), hotaDir)
	return known
}

// startHotaPoller polls Games/HotA Random/ every 60 seconds for new XXX.GM2
// files. New files (not in the initial snapshot) are uploaded with a 3-second debounce
func (a *App) startHotaPoller(root string, stopPoll <-chan struct{}) {
	hotaDir := filepath.Join(root, "Games", "HotA Random")
	known := snapshotHotaFiles(hotaDir)

	type pending struct {
		absPath string
		timer   *time.Timer
	}
	debounce := make(map[string]*pending)
	var dmu sync.Mutex

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	log.Printf("hota poller started, watching: %s", hotaDir)

	for {
		select {
		case <-stopPoll:
			log.Println("hota poller stopped")
			return
		case <-ticker.C:
			err := filepath.WalkDir(hotaDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if !endTurnFile.MatchString(d.Name()) {
					return nil
				}
				rel, _ := filepath.Rel(hotaDir, path)
				if _, exists := known[rel]; exists {
					return nil
				}
				// New file — add to known and start debounce.
				known[rel] = struct{}{}
				absPath, _ := filepath.Abs(path)
				log.Printf("hota poller: new file detected: %s", rel)
				dmu.Lock()
				if p, exists := debounce[absPath]; exists {
					p.timer.Reset(3 * time.Second)
				} else {
					p := &pending{absPath: absPath}
					p.timer = time.AfterFunc(3*time.Second, func() {
						dmu.Lock()
						delete(debounce, absPath)
						dmu.Unlock()
						a.uploadFile(absPath, "TURN_END")
					})
					debounce[absPath] = p
				}
				dmu.Unlock()
				return nil
			})
			if err != nil {
				log.Printf("hota poller walk error: %v", err)
			}
		}
	}
}
