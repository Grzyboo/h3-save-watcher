package main

import (
	"context"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/fsnotify/fsnotify"
)

// App holds all application state.
type App struct {
	mu               sync.Mutex
	bus              *Bus
	lang             Lang
	firstRun         bool
	watchDir         string
	instanceID       string
	watcher          *fsnotify.Watcher
	lastUploadedHash map[string]string // keyed by fileType
	gameInfo         GameInfo
	logs             []LogEntry
	logList          *widget.List
	dirLabel         *widget.Label
	selectedLabel    *widget.Label
	browseBtn        *widget.Button
	startupBtn       *widget.Button
	window           fyne.Window

	// Game folder watching
	watchedGameFolder  string
	gameFolderCancel   context.CancelFunc
	gameFolderDebounce *time.Timer

	// True while the fsnotify watcher has an active watch on <root>/Games.
	// The root dir is always watched so that a later-created (or deleted and
	// re-created) Games folder can be picked up without restarting the app.
	gamesDirWatched bool

	// Persisted list of files already sent from each game folder.
	sentFoldersCache SentFoldersCache
	sentFoldersMu    sync.Mutex

	// True on the very first program run; existing folder contents are marked
	// as sent without uploading, so the user is not flooded with old saves.
	isInitialRun bool

	// Active upload tracking (used to wait for uploads before auto-restart).
	uploadMu    sync.Mutex
	uploadCond  *sync.Cond
	uploadCount int
}

func (a *App) refreshStartupBtn() {
	if a.startupBtn == nil {
		return
	}
	if isStartupEnabled() {
		a.startupBtn.SetText(a.T(KeyStartupDisable))
	} else {
		a.startupBtn.SetText(a.T(KeyStartupEnable))
	}
}
