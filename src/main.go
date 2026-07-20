package main

import (
	"log"
	"os"
	"path/filepath"
	"sync"
)

func main() {
	_ = ensureAppDir()

	// Debug log file — written to the dedicated application directory.
	logPath := filepath.Join(ensureAppDir(), "h3savewatcher.log")
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644); err == nil {
		log.SetOutput(f)
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		defer f.Close()
	}
	log.Println("startup args:", os.Args)

	cleanOldBinary()
	migrateLegacyConfigAndCache()

	startInTray := len(os.Args) > 1 && os.Args[1] == "--tray"
	log.Println("startInTray:", startInTray)

	fyneApp, w := newFyneApp("H3SaveWatcher")
	log.Println("fyne app created")

	cfg := loadConfig()
	// If the config file already existed before this launch, treat it as an
	// existing installation even if the InitialRunCompleted flag is missing
	// (e.g. upgrade from an older version).
	_, configExistedErr := os.Stat(configPath())
	configExisted := configExistedErr == nil
	ensureInstanceID(&cfg)

	isInitialRun := !cfg.InitialRunCompleted && !configExisted
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
