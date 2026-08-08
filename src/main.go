package main

import (
	"log"
	"os"
	"path/filepath"
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

	fyneApp, w := newFyneApp("Ajit")
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

	bus := NewBus()
	go bus.Run()
	defer bus.Stop()

	s := &State{
		isInitialRun:     isInitialRun,
		lastUploadedHash: make(map[string]string),
		sentFoldersCache: loadSentFoldersCache(),
		instanceID:       cfg.InstanceID,
	}
	s.setLang(startLang)
	log.Printf("instance ID: %s", cfg.InstanceID)

	logs := &logStore{}
	watcher := NewWatcher(bus)
	sched := newGameFolderScheduler()
	uploads := newUploadTracker()
	ui := &uiRefs{window: w}

	registerPasswordsHandlers(bus, s)
	registerUploadHandlers(bus, s, uploads)
	registerGameFolderHandlers(bus, s, watcher, sched)
	registerWatchDirHandlers(bus, s, ui, watcher)
	registerLanguageHandlers(bus, s, ui, logs)
	registerStartupHandlers(bus, s, ui)
	registerLogHandlers(bus, s, logs)
	registerOnboardingHandlers(bus, s, ui, watcher)

	up := &updater{bus: bus, watcher: watcher, sched: sched, uploads: uploads}
	// Check for updates every hour in background — fails silently, app keeps running.
	go up.startUpdateChecker()

	buildUI(bus, s, ui, logs, w)
	up.showUpdateNotification()
	logs.startPruner()

	// Onboarding decision:
	// - no config at all -> onboarding
	// - config with onboarding_finished -> skip
	// - config without onboarding_finished -> skip only when a watch_dir is
	//   already configured (existing user upgrading from an older version)
	needsOnboarding := !cfg.OnboardingFinished && cfg.WatchDir == ""
	if needsOnboarding {
		ob := newOnboarding(bus, s, w)
		w.SetContent(ob.Root())
	} else {
		w.SetContent(ui.mainContent)
		if cfg.WatchDir != "" {
			if root, err := resolveH3Root(cfg.WatchDir); err == nil {
				if root != cfg.WatchDir {
					cfg.WatchDir = root
					saveConfig(cfg)
				}
				ui.dirLabel.SetText(root)
				watcher.Start(root)
			}
		}
	}

	log.Println("pre-tray setup")
	setupTray(s, w, fyneApp)

	// Hide to tray on window close instead of quitting.
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	log.Println("entering event loop, startInTray:", startInTray, "needsOnboarding:", needsOnboarding)
	if startInTray && !needsOnboarding {
		w.Hide()
		log.Println("window hidden, calling fyneApp.Run()")
		fyneApp.Run()
	} else {
		// Onboarding takes precedence over a tray start: the user must
		// finish the wizard before the main application is usable.
		w.ShowAndRun()
	}

	_ = watcher.Close()
	sched.stop()
}
