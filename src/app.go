package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/fsnotify/fsnotify"
)

// App holds all application state.
type App struct {
	mu               sync.Mutex
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

func (a *App) setLang(lang Lang) {
	a.mu.Lock()
	a.lang = lang
	a.mu.Unlock()

	cfg := loadConfig()
	cfg.Language = string(lang)
	saveConfig(cfg)

	fyne.Do(func() {
		a.selectedLabel.SetText(a.T(KeySelectedFolder))
		a.browseBtn.SetText(a.T(KeyChooseDirectory))
		hasDirSet := a.dirLabel.Text == "" || a.dirLabel.Text == a.T(KeyNoDirectorySelected)
		if hasDirSet {
			a.dirLabel.SetText(a.T(KeyNoDirectorySelected))
		}
		a.refreshStartupBtn()
		if a.logList != nil {
			a.logList.Refresh()
		}
	})
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

func (a *App) handleStartupToggle() {
	enabled := isStartupEnabled()

	if !enabled {
		// Check for suspicious path before enabling.
		exe, err := os.Executable()
		if err == nil {
			exe, _ = filepath.EvalSymlinks(exe)
		}
		if isSuspiciousPath(exe) {
			dialog.ShowConfirm(
				a.T(KeyStartupWarnTitle),
				fmt.Sprintf(a.T(KeyStartupWarnMsg), exe),
				func(proceed bool) {
					if proceed {
						a.confirmEnableStartup()
					}
				},
				a.window,
			)
			return
		}
		a.confirmEnableStartup()
	} else {
		dialog.ShowConfirm(
			a.T(KeyStartupDisableTitle),
			a.T(KeyStartupDisableConfirm),
			func(ok bool) {
				if !ok {
					return
				}
				if err := disableStartup(); err != nil {
					dialog.ShowError(fmt.Errorf(a.T(KeyStartupError), err), a.window)
					return
				}
				a.refreshStartupBtn()
				dialog.ShowInformation(a.T(KeyStartupDisableTitle), a.T(KeyStartupRemoved), a.window)
			},
			a.window,
		)
	}
}

func (a *App) confirmEnableStartup() {
	dialog.ShowConfirm(
		a.T(KeyStartupEnableTitle),
		a.T(KeyStartupEnableConfirm),
		func(ok bool) {
			if !ok {
				return
			}
			if err := enableStartup(); err != nil {
				dialog.ShowError(fmt.Errorf(a.T(KeyStartupError), err), a.window)
				return
			}
			a.refreshStartupBtn()
			dialog.ShowInformation(a.T(KeyStartupEnableTitle), a.T(KeyStartupSuccess), a.window)
		},
		a.window,
	)
}

func (a *App) setWatchDir(dir string) {
	root, err := resolveH3Root(dir)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}
	a.dirLabel.SetText(root)
	cfg := loadConfig()
	cfg.WatchDir = root
	saveConfig(cfg)
	a.startWatcher(root)

	if a.firstRun && !isStartupEnabled() {
		a.firstRun = false
		a.handleStartupToggle()
	}
}
