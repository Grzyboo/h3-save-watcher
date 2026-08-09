package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	cacheFile                  = "h3savewatcher_cache.json"
	maxSentFolders             = 20
	legacyInitialRunInfoPrefix = "Initial run, didn't send files: "
)

// SentFoldersCache persists the list of game folders and the files that have
// already been uploaded from them. It is used to avoid re-uploading the whole
// folder when the program is restarted.
type SentFoldersCache struct {
	Folders []SentFolder `json:"folders"`
}

// SentFolder holds the list of sent files for a single game folder.
type SentFolder struct {
	Path  string   `json:"path"`
	Files []string `json:"files"`
	Info  string   `json:"info,omitempty"` // legacy initial-run migration marker
}

// legacyCachePath returns the old cache path directly in the user's config
// directory, used for one-time migration.
func legacyCachePath() string {
	return filepath.Join(userConfigDir(), cacheFile)
}

func cachePath() string {
	return filepath.Join(ensureAppDir(), cacheFile)
}

// tmpCachePath returns the path used for atomic cache writes.
func tmpCachePath() string {
	return cachePath() + ".tmp"
}

// loadSentFoldersCache reads the persisted cache or returns an empty one.
func loadSentFoldersCache() SentFoldersCache {
	// Clean up any leftover temporary file from an interrupted atomic write.
	if err := os.Remove(tmpCachePath()); err != nil && !os.IsNotExist(err) {
		log.Printf("failed to remove stale cache temp file: %v", err)
	}

	data, err := os.ReadFile(cachePath())
	if err != nil {
		return SentFoldersCache{}
	}
	var cache SentFoldersCache
	if err := json.Unmarshal(data, &cache); err != nil {
		log.Printf("failed to parse sent folders cache: %v", err)
		return SentFoldersCache{}
	}
	if cache.migrateLegacyInitialRun() {
		if err := cache.save(); err != nil {
			log.Printf("failed to save migrated sent folders cache: %v", err)
		}
	}
	return cache
}

// migrateLegacyInitialRun removes files that an older version marked as sent
// without uploading during its first run. Successful uploads recorded after
// that scan remain cached.
func (c *SentFoldersCache) migrateLegacyInitialRun() bool {
	changed := false
	for i := range c.Folders {
		info := c.Folders[i].Info
		if !strings.HasPrefix(info, legacyInitialRunInfoPrefix) {
			continue
		}

		initialFiles := strings.Split(strings.TrimPrefix(info, legacyInitialRunInfoPrefix), ", ")
		initialSet := make(map[string]struct{}, len(initialFiles))
		for _, filename := range initialFiles {
			initialSet[filename] = struct{}{}
		}

		kept := c.Folders[i].Files[:0]
		for _, filename := range c.Folders[i].Files {
			if _, wasSkipped := initialSet[filename]; !wasSkipped {
				kept = append(kept, filename)
			}
		}
		c.Folders[i].Files = kept
		c.Folders[i].Info = ""
		changed = true
	}
	return changed
}

// save writes the cache to disk atomically (temp file + rename).
func (c *SentFoldersCache) save() error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	tmp := tmpCachePath()
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, cachePath())
}

// moveToFront moves the folder at idx to the front of the list.
func (c *SentFoldersCache) moveToFront(idx int) {
	if idx == 0 || idx >= len(c.Folders) {
		return
	}
	folder := c.Folders[idx]
	copy(c.Folders[1:], c.Folders[:idx])
	c.Folders[0] = folder
}

// touchFolder moves the folder to the front if it exists. It does not create
// a new entry.
func (c *SentFoldersCache) touchFolder(path string) {
	for i, f := range c.Folders {
		if f.Path == path {
			c.moveToFront(i)
			return
		}
	}
}

// ensureFolder makes sure the folder is in the cache at the front. If it is
// missing, a new empty entry is created.
func (c *SentFoldersCache) ensureFolder(path string) {
	for i, f := range c.Folders {
		if f.Path == path {
			c.moveToFront(i)
			return
		}
	}
	c.Folders = append([]SentFolder{{Path: path, Files: []string{}}}, c.Folders...)
	if len(c.Folders) > maxSentFolders {
		c.Folders = c.Folders[:maxSentFolders]
	}
}

// addFile marks a file as sent in the given folder. The folder is moved to the
// front of the list. Duplicate filenames are ignored.
func (c *SentFoldersCache) addFile(path, filename string) {
	c.ensureFolder(path)
	for _, f := range c.Folders[0].Files {
		if f == filename {
			return
		}
	}
	c.Folders[0].Files = append(c.Folders[0].Files, filename)
}

// hasFile reports whether a file was already sent from the given folder.
func (c *SentFoldersCache) hasFile(path, filename string) bool {
	for _, f := range c.Folders {
		if f.Path != path {
			continue
		}
		for _, file := range f.Files {
			if file == filename {
				return true
			}
		}
		return false
	}
	return false
}
