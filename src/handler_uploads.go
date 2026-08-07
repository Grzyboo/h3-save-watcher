package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// uploadTracker is the upload counter/condition used by the auto-updater.
// Upload state is changed on the bus goroutine; the mutex protects reads from
// the updater goroutine while it waits for a restart-safe point.
type uploadTracker struct {
	mu    sync.Mutex
	cond  *sync.Cond
	count int
}

func newUploadTracker() *uploadTracker {
	t := &uploadTracker{}
	t.cond = sync.NewCond(&t.mu)
	return t
}

func (t *uploadTracker) begin() {
	t.mu.Lock()
	t.count++
	t.mu.Unlock()
}

func (t *uploadTracker) done() {
	t.mu.Lock()
	t.count--
	if t.count < 0 {
		t.count = 0
	}
	t.cond.Broadcast()
	t.mu.Unlock()
}

// wait blocks until all active uploads finish or the timeout expires.
func (t *uploadTracker) wait(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		t.mu.Lock()
		for t.count > 0 {
			t.cond.Wait()
		}
		t.mu.Unlock()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// registerUploadHandlers wires the upload pipeline: the five save-detected
// events -> hash dedup -> upload goroutine -> result events. The slow network
// work runs on spawned goroutines so the event loop stays responsive and
// uploads stay concurrent.
func registerUploadHandlers(bus *Bus, s *State, uploads *uploadTracker) {
	detect := func(path, fileType string) {
		bus.Publish(UploadStarted{Path: path})

		data, err := os.ReadFile(path)
		if err != nil {
			bus.Publish(UploadFailed{Path: path, Kind: UploadErrRead, Err: err})
			return
		}

		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		if s.lastUploadedHash[fileType] == hash {
			log.Printf("skipping duplicate upload: %s (%s), hash %s already uploaded", filepath.Base(path), fileType, hash)
			bus.Publish(UploadSkippedDuplicate{Path: path})
			return
		}
		s.lastUploadedHash[fileType] = hash
		info := s.gameInfo
		instanceID := s.instanceID

		go func() {
			outcome := uploadSaveFile(path, data, info, instanceID)
			switch outcome.result {
			case uploadOK:
				bus.Publish(UploadSucceeded{Path: path})
			case uploadAlreadyExists:
				bus.Publish(UploadAlreadyOnServer{Path: path})
			default:
				bus.Publish(UploadFailed{Path: path, Kind: outcome.kind, Err: outcome.err})
			}
		}()
	}

	Subscribe(bus, func(e BattleSaveDetected) { detect(e.Path, "BATTLE") })
	Subscribe(bus, func(e BattleNonContinuableSaveDetected) { detect(e.Path, "BATTLE_NC") })
	Subscribe(bus, func(e TurnBeginSaveDetected) { detect(e.Path, "TURN_BEGIN") })
	Subscribe(bus, func(e GameBeginSaveDetected) { detect(e.Path, "GAME_BEGIN") })
	Subscribe(bus, func(e TurnEndSaveDetected) { detect(e.Path, "TURN_END") })

	// inFlight tracks started uploads by path (bus goroutine only) so that an
	// UploadFailed without a matching UploadStarted, such as a folder scan
	// read error, never touches the counter.
	inFlight := make(map[string]int)
	Subscribe(bus, func(e UploadStarted) {
		inFlight[e.Path]++
		uploads.begin()
	})

	finish := func(path string) {
		if inFlight[path] == 0 {
			return
		}
		inFlight[path]--
		if inFlight[path] == 0 {
			delete(inFlight, path)
		}
		uploads.done()
	}

	Subscribe(bus, func(e UploadSucceeded) {
		finish(e.Path)
		markFileAsSent(s, e.Path)
	})
	Subscribe(bus, func(e UploadAlreadyOnServer) {
		finish(e.Path)
		log.Printf("upload %s: file already exists on server, cached as sent", filepath.Base(e.Path))
		markFileAsSent(s, e.Path)
	})
	Subscribe(bus, func(e UploadSkippedDuplicate) { finish(e.Path) })
	Subscribe(bus, func(e UploadFailed) { finish(e.Path) })
}

// markFileAsSent records a file in the sent-folders cache if it belongs to
// the currently watched game folder, so the app does not re-upload it after
// a restart.
func markFileAsSent(s *State, path string) {
	folder := s.watchedGameFolder
	if folder == "" {
		return
	}

	absPath, _ := filepath.Abs(path)
	absFolder, _ := filepath.Abs(folder)
	if !strings.HasPrefix(absPath, absFolder+string(os.PathSeparator)) {
		return
	}

	s.sentFoldersCache.addFile(absFolder, filepath.Base(path))
	if err := s.sentFoldersCache.save(); err != nil {
		log.Printf("failed to save sent folders cache: %v", err)
	}
}
