package main

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// State holds all application state. It is owned by the bus goroutine:
// handlers may mutate it freely without locks. The only exception is lang,
// which is also read from Fyne's thread (log list rendering) via T, so it is
// guarded by its own small mutex.
type State struct {
	watchDir          string
	gameInfo          GameInfo
	watchedGameFolder string
	gamesDirWatched   bool
	isInitialRun      bool
	lastUploadedHash  map[string]string // keyed by fileType
	sentFoldersCache  SentFoldersCache
	instanceID        string

	langMu sync.RWMutex
	lang   Lang
}

// T translates a key into the current language, falling back to English.
func (s *State) T(key TranslationKey) string {
	s.langMu.RLock()
	lang := s.lang
	s.langMu.RUnlock()
	if t, ok := translations[lang]; ok {
		if str, ok := t[key]; ok {
			return str
		}
	}
	// fallback to English
	if str, ok := translations[LangEN][key]; ok {
		return str
	}
	return string(key)
}

func (s *State) setLang(lang Lang) {
	s.langMu.Lock()
	s.lang = lang
	s.langMu.Unlock()
}

// Lang returns the current language.
func (s *State) Lang() Lang {
	s.langMu.RLock()
	defer s.langMu.RUnlock()
	return s.lang
}

// uiRefs holds the widget pointers needed by UI-touching handlers. They are
// assigned once during UI construction (buildUI); any widget access from a
// handler is wrapped in fyne.Do.
type uiRefs struct {
	window        fyne.Window
	mainContent   fyne.CanvasObject
	dirLabel      *widget.Label
	selectedLabel *widget.Label

	// Settings panel refs; settingsDirLabel is rebuilt on each open.
	settingsDirLabel *widget.Label
	startupCheck     *widget.Check
	showingSettings  bool
}
