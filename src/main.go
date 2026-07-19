package main

import (
	"log"
	"os"
	"path/filepath"
	"sync"
)

func main() {
	// Debug log file — written next to the executable.
	if exePath, err := os.Executable(); err == nil {
		logPath := filepath.Join(filepath.Dir(exePath), "h3savewatcher.log")
		if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err == nil {
			log.SetOutput(f)
			log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
			defer f.Close()
		}
	}
	log.Println("startup args:", os.Args)

	cleanOldBinary()

	startInTray := len(os.Args) > 1 && os.Args[1] == "--tray"
	log.Println("startInTray:", startInTray)

	fyneApp, w := newFyneApp("H3SaveWatcher")
	log.Println("fyne app created")

	cfg := loadConfig()
	ensureInstanceID(&cfg)
	isInitialRun := !cfg.InitialRunCompleted
	if isInitialRun {
		cfg.InitialRunCompleted = true
		saveConfig(cfg)
	}
	startLang := Lang(cfg.Language)
	if startLang == "" {
		startLang = LangEN
	}

	state := &App{window: w, lang: startLang, firstRun: cfg.WatchDir == "", instanceID: cfg.InstanceID, lastUploadedHash: make(map[string]string), sentFoldersCache: loadSentFoldersCache(), isInitialRun: isInitialRun}
	state.uploadCond = sync.NewCond(&state.uploadMu)
	log.Printf("instance ID: %s", cfg.InstanceID)

	// Check for updates every hour in background — fails silently, app keeps running.
	go state.startUpdateChecker()

	buildUI(state, w, fyneApp)
	state.showUpdateNotification()
	state.startLogPruner()

	if cfg.WatchDir != "" {
		if root, err := resolveH3Root(cfg.WatchDir); err == nil {
			state.dirLabel.SetText(root)
			state.startWatcher(root)
		}
	} else if !startInTray {
		// First run — attempt auto-discovery after window is shown.
		// Skip dialogs when starting in tray; user can open the window manually.
		runAutoDiscovery(state, w)
	}

	log.Println("pre-tray setup")
	setupTray(state, w, fyneApp)

	// Hide to tray on window close instead of quitting.
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	log.Println("entering event loop, startInTray:", startInTray)
	if startInTray {
		w.Hide()
		log.Println("window hidden, calling fyneApp.Run()")
		fyneApp.Run()
	} else {
		w.ShowAndRun()
	}

	state.mu.Lock()
	if state.watcher != nil {
		state.watcher.Close()
	}
	if state.gameFolderCancel != nil {
		state.gameFolderCancel()
		state.gameFolderCancel = nil
	}
	if state.gameFolderDebounce != nil {
		state.gameFolderDebounce.Stop()
		state.gameFolderDebounce = nil
	}
	state.mu.Unlock()
}
