package main

import (
	"fmt"
	"image/color"
	"log"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	logMaxAge        = 24 * time.Hour
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

// logStore owns the activity log. The bus appends entries while widget.List
// callbacks and the pruner may read them from other goroutines.
type logStore struct {
	mu      sync.Mutex
	entries []LogEntry
	list    *widget.List
}

func (l *logStore) add(success bool, key TranslationKey, args ...any) {
	entry := LogEntry{Time: time.Now(), Success: success, Key: key, Args: args}
	l.mu.Lock()
	l.entries = append(l.entries, entry)
	n := len(l.entries)
	list := l.list
	l.mu.Unlock()
	if list == nil {
		return
	}
	fyne.Do(func() {
		list.Refresh()
		if n > 0 {
			list.ScrollTo(n - 1)
		}
	})
}

func (l *logStore) setList(list *widget.List) {
	l.mu.Lock()
	l.list = list
	l.mu.Unlock()
}

// refresh is called on Fyne's goroutine by UI handlers.
func (l *logStore) refresh() {
	l.mu.Lock()
	list := l.list
	l.mu.Unlock()
	if list != nil {
		list.Refresh()
	}
}

func (l *logStore) startPruner() {
	go func() {
		ticker := time.NewTicker(logPruneInterval)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-logMaxAge)
			l.mu.Lock()
			before := len(l.entries)
			keep := l.entries[:0]
			for _, e := range l.entries {
				if !e.Time.Before(cutoff) {
					keep = append(keep, e)
				}
			}
			l.entries = keep
			removed := before - len(l.entries)
			l.mu.Unlock()
			if removed > 0 {
				log.Printf("log pruner: removed %d entr%s older than 24h", removed, map[bool]string{true: "y", false: "ies"}[removed == 1])
				fyne.Do(func() { l.refresh() })
			} else {
				log.Println("log pruner: ran, no stale entries found")
			}
		}
	}()
}

func buildLogList(store *logStore, translate func(TranslationKey) string) *widget.List {
	var list *widget.List
	list = widget.NewList(
		func() int {
			store.mu.Lock()
			defer store.mu.Unlock()
			return len(store.entries)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			store.mu.Lock()
			entry := store.entries[id]
			store.mu.Unlock()
			lbl := obj.(*widget.Label)
			lbl.SetText(entry.Format(translate(entry.Key)))
			if entry.Success {
				lbl.Importance = widget.MediumImportance
			} else {
				lbl.Importance = widget.DangerImportance
			}
			list.SetItemHeight(id, lbl.MinSize().Height)
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		list.Unselect(id)
	}
	store.setList(list)
	return list
}

func buildLogPanel(store *logStore, translate func(TranslationKey) string) fyne.CanvasObject {
	list := buildLogList(store, translate)
	bg := canvas.NewRectangle(color.NRGBA{R: 30, G: 30, B: 30, A: 255})
	padded := container.NewPadded(list)
	return container.NewPadded(container.NewStack(bg, padded))
}
