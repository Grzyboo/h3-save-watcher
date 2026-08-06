package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// registerUploadHandlers wires the upload pipeline: the five save-detected
// events → hash dedup → upload goroutine → result events. The slow network
// work runs on spawned goroutines so the event loop stays responsive and
// uploads stay concurrent; results are reported back as events.
//
// It also owns the upload counter used by the auto-updater to wait for
// in-flight uploads before restarting: UploadStarted increments it, the
// terminal events (UploadSucceeded/UploadAlreadyOnServer/
// UploadSkippedDuplicate/UploadFailed) decrement it.
func registerUploadHandlers(bus *Bus, a *App) {
	detect := func(path, fileType string) {
		bus.Publish(UploadStarted{Path: path})

		data, err := os.ReadFile(path)
		if err != nil {
			bus.Publish(UploadFailed{Path: path, Kind: UploadErrRead, Err: err})
			return
		}

		hash := fmt.Sprintf("%x", sha256.Sum256(data))
		a.mu.Lock()
		if a.lastUploadedHash[fileType] == hash {
			a.mu.Unlock()
			log.Printf("skipping duplicate upload: %s (%s), hash %s already uploaded", filepath.Base(path), fileType, hash)
			bus.Publish(UploadSkippedDuplicate{Path: path})
			return
		}
		a.lastUploadedHash[fileType] = hash
		info := a.gameInfo
		a.mu.Unlock()

		go func() {
			outcome := uploadSaveFile(path, data, info, a.instanceID)
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
	// UploadFailed without a matching UploadStarted (e.g. a folder scan read
	// error) never touches the counter.
	inFlight := make(map[string]int)

	Subscribe(bus, func(e UploadStarted) {
		inFlight[e.Path]++
		a.uploadMu.Lock()
		a.uploadCount++
		a.uploadMu.Unlock()
	})

	finish := func(path string) {
		if inFlight[path] == 0 {
			return
		}
		inFlight[path]--
		if inFlight[path] == 0 {
			delete(inFlight, path)
		}
		a.uploadMu.Lock()
		a.uploadCount--
		a.uploadCond.Broadcast()
		a.uploadMu.Unlock()
	}

	Subscribe(bus, func(e UploadSucceeded) {
		finish(e.Path)
		a.markFileAsSent(e.Path)
	})
	Subscribe(bus, func(e UploadAlreadyOnServer) {
		finish(e.Path)
		log.Printf("upload %s: file already exists on server, cached as sent", filepath.Base(e.Path))
		a.markFileAsSent(e.Path)
	})
	Subscribe(bus, func(e UploadSkippedDuplicate) { finish(e.Path) })
	Subscribe(bus, func(e UploadFailed) { finish(e.Path) })
}

// markFileAsSent records a file in the sent-folders cache if it belongs to
// the currently watched game folder, so the app does not re-upload it after
// a restart.
func (a *App) markFileAsSent(path string) {
	a.mu.Lock()
	folder := a.watchedGameFolder
	a.mu.Unlock()
	if folder == "" {
		return
	}

	absPath, _ := filepath.Abs(path)
	absFolder, _ := filepath.Abs(folder)
	if !strings.HasPrefix(absPath, absFolder+string(os.PathSeparator)) {
		return
	}

	a.sentFoldersMu.Lock()
	a.sentFoldersCache.addFile(absFolder, filepath.Base(path))
	if err := a.sentFoldersCache.save(); err != nil {
		log.Printf("failed to save sent folders cache: %v", err)
	}
	a.sentFoldersMu.Unlock()
}
