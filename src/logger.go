package main

import (
	"fmt"
	"image/color"
	"log"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	logMaxAge       = 24 * time.Hour
	logPruneInterval = time.Hour
)

// LogEntry represents a single activity log item.
type LogEntry struct {
	Time    time.Time
	Success bool
	Key     TranslationKey
	Args    []any
}

func (e LogEntry) Format(t string) string {
	status := ""
	if !e.Success {
		status = "ERR "
	}
	msg := t
	if len(e.Args) > 0 {
		msg = fmt.Sprintf(t, e.Args...)
	}
	return fmt.Sprintf("[%s] %s%s", e.Time.Format("15:04:05"), status, msg)
}

func (a *App) addLog(success bool, key TranslationKey, args ...any) {
	entry := LogEntry{Time: time.Now(), Success: success, Key: key, Args: args}
	a.mu.Lock()
	a.logs = append(a.logs, entry)
	n := len(a.logs)
	a.mu.Unlock()
	fyne.Do(func() {
		a.logList.Refresh()
		if n > 0 {
			a.logList.ScrollTo(n - 1)
		}
	})
}

func (a *App) relPath(abs string) string {
	return filepath.Base(abs)
}

func buildLogList(a *App) *widget.List {
	list := widget.NewList(
		func() int {
			a.mu.Lock()
			defer a.mu.Unlock()
			return len(a.logs)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			a.mu.Lock()
			entry := a.logs[id]
			a.mu.Unlock()
			lbl := obj.(*widget.Label)
			lbl.SetText(entry.Format(a.T(entry.Key)))
			if entry.Success {
				lbl.Importance = widget.MediumImportance
			} else {
				lbl.Importance = widget.DangerImportance
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		list.Unselect(id)
	}
	a.logList = list
	return list
}

func (a *App) startLogPruner() {
	go func() {
		ticker := time.NewTicker(logPruneInterval)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-logMaxAge)
			a.mu.Lock()
			before := len(a.logs)
			keep := a.logs[:0]
			for _, e := range a.logs {
				if !e.Time.Before(cutoff) {
					keep = append(keep, e)
				}
			}
			a.logs = keep
			removed := before - len(a.logs)
			a.mu.Unlock()
			if removed > 0 {
				log.Printf("log pruner: removed %d entr%s older than 24h", removed, map[bool]string{true: "y", false: "ies"}[removed == 1])
				fyne.Do(func() { a.logList.Refresh() })
			} else {
				log.Println("log pruner: ran, no stale entries found")
			}
		}
	}()
}

func buildLogPanel(a *App) fyne.CanvasObject {
	list := buildLogList(a)
	bg := canvas.NewRectangle(color.NRGBA{R: 30, G: 30, B: 30, A: 255})
	padded := container.NewPadded(list)
	return container.NewPadded(container.NewStack(bg, padded))
}
