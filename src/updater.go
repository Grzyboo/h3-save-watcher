package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	updateCheckTimeout  = 30 * time.Second
	updateDownTimeout   = 5 * time.Minute
	updateCheckInterval = 1 * time.Hour
	uploadWaitTimeout   = 60 * time.Second
)

// githubRelease is the subset of the GitHub releases API response we care about.
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// updater owns the self-update workflow and the producer references needed
// to stop watching and wait for active uploads before restarting.
type updater struct {
	bus     *Bus
	watcher *Watcher
	sched   *gameFolderScheduler
	uploads *uploadTracker
}

// cleanOldBinary removes <exe>.old left behind by a previous self-replace on
// Windows (where the running binary cannot be deleted while open).
func cleanOldBinary() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	old := exe + ".old"
	if _, err := os.Stat(old); err == nil {
		if err := os.Remove(old); err != nil {
			log.Printf("updater: could not remove old binary %s: %v", old, err)
		} else {
			log.Printf("updater: removed old binary %s", old)
		}
	}
}

func (u *updater) startUpdateChecker() {
	for {
		u.checkAndUpdate()
		time.Sleep(updateCheckInterval)
	}
}

func (u *updater) stopWatcher() {
	if err := u.watcher.Close(); err != nil {
		log.Printf("updater: error closing watcher: %v", err)
	}
	u.sched.stop()
	log.Println("updater: stopped file watcher before restart")
}

// checks for a newer release, downloads + verifies it, self-replaces the binary, then restarts the process.
// Any failure is logged and the function returns silently — the app keeps running.
func (u *updater) checkAndUpdate() {
	if githubRepo == "" || appVersion == "" {
		log.Println("updater: githubRepo or appVersion not set, skipping")
		return
	}

	if strings.Contains(appVersion, "SNAPSHOT") {
		log.Println("updater: SNAPSHOT version detected, skipping auto-update")
		return
	}

	log.Printf("updater: current version %s, checking %s", appVersion, githubRepo)

	release, err := fetchLatestRelease(githubRepo)
	if err != nil {
		log.Printf("updater: failed to fetch release info: %v", err)
		return
	}

	if !isNewer(release.TagName, appVersion) {
		log.Printf("updater: already up to date (%s)", appVersion)
		return
	}

	log.Printf("updater: new version available: %s", release.TagName)

	assetName := expectedAssetName()
	assetURL := findAssetURL(release.Assets, assetName)
	if assetURL == "" {
		log.Printf("updater: no asset named %q found in release %s", assetName, release.TagName)
		return
	}

	checksumURL := findAssetURL(release.Assets, "checksums.txt")

	// Download new binary to a temp file in the same directory as the
	// running executable (same filesystem → atomic rename possible).
	exe, err := os.Executable()
	if err != nil {
		log.Printf("updater: cannot determine executable path: %v", err)
		return
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		log.Printf("updater: cannot resolve symlinks: %v", err)
		return
	}
	exeDir := filepath.Dir(exe)

	tmpFile, err := os.CreateTemp(exeDir, "h3savewatcher-update-*")
	if err != nil {
		log.Printf("updater: cannot create temp file: %v", err)
		return
	}
	tmpPath := tmpFile.Name()

	// Clean up temp on any failure path.
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	log.Printf("updater: downloading %s", assetURL)
	downloaded, err := downloadToFile(assetURL, tmpFile)
	tmpFile.Close()
	if err != nil {
		log.Printf("updater: download failed: %v", err)
		return
	}

	// Verify checksum if checksums.txt is present in the release.
	if checksumURL != "" {
		expected, err := fetchChecksum(checksumURL, assetName)
		if err != nil {
			log.Printf("updater: checksum fetch failed: %v", err)
			return
		}
		actual, err := sha256File(tmpPath)
		if err != nil {
			log.Printf("updater: sha256 of download failed: %v", err)
			return
		}
		if !strings.EqualFold(actual, expected) {
			log.Printf("updater: checksum mismatch — expected %s got %s", expected, actual)
			return
		}
		log.Printf("updater: checksum OK (%s)", actual)
	} else {
		log.Printf("updater: no checksums.txt in release, skipping verification (downloaded %d bytes)", downloaded)
	}

	// Make executable on Unix.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			log.Printf("updater: chmod failed: %v", err)
			return
		}
	}

	// Self-replace:
	//   Windows: rename running exe → exe.old (can't delete while running)
	//            then rename tmp → exe
	//   Unix:    rename tmp → exe (atomic, replaces in-place)
	if runtime.GOOS == "windows" {
		oldPath := exe + ".old"
		_ = os.Remove(oldPath) // remove any stale .old from a prior update
		if err := os.Rename(exe, oldPath); err != nil {
			log.Printf("updater: rename current→old failed: %v", err)
			return
		}
	}
	if err := os.Rename(tmpPath, exe); err != nil {
		log.Printf("updater: rename tmp→exe failed: %v", err)
		// On Windows the old binary was already renamed; try to restore it.
		if runtime.GOOS == "windows" {
			_ = os.Rename(exe+".old", exe)
		}
		return
	}

	success = true
	log.Printf("updater: updated to %s, preparing to restart", release.TagName)

	// Stop watching for further file changes, then wait briefly for any
	// in-progress uploads to finish before restarting.
	u.stopWatcher()
	if u.uploads.wait(uploadWaitTimeout) {
		log.Println("updater: all active uploads finished, restarting")
	} else {
		log.Printf("updater: timed out waiting for uploads after %v, restarting anyway", uploadWaitTimeout)
	}

	// Restart: launch a new copy of the (now-updated) binary with the same arguments, then exit this process.
	restartSelf(exe, release.TagName)
}

func fetchLatestRelease(repo string) (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	client := &http.Client{Timeout: updateCheckTimeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Ajit/"+appVersion)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	if release.TagName == "" {
		return nil, fmt.Errorf("release tag_name is empty")
	}
	return &release, nil
}

// expectedAssetName returns the asset filename for the current OS+arch.
func expectedAssetName() string {
	name := fmt.Sprintf("h3savewatcher-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func findAssetURL(assets []githubAsset, name string) string {
	for _, a := range assets {
		if strings.EqualFold(a.Name, name) {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

func downloadToFile(url string, dst *os.File) (int64, error) {
	client := &http.Client{Timeout: updateDownTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("download returned status %d", resp.StatusCode)
	}
	return io.Copy(dst, resp.Body)
}

// fetchChecksum downloads checksums.txt and returns the SHA256 hex string for the given asset filename.
func fetchChecksum(url, assetName string) (string, error) {
	client := &http.Client{Timeout: updateCheckTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums.txt download returned status %d", resp.StatusCode)
	}

	// Format written by sha256sum: "<hash>  <filename>\n"
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.EqualFold(parts[1], assetName) {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("no entry for %q in checksums.txt", assetName)
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// isNewer returns true when latest > current.
// Expects semver strings like "v1.2.3". Falls back to string comparison.
func isNewer(latest, current string) bool {
	lv := parseSemver(latest)
	cv := parseSemver(current)
	for i := 0; i < 3; i++ {
		if lv[i] > cv[i] {
			return true
		}
		if lv[i] < cv[i] {
			return false
		}
	}
	return false
}

func parseSemver(s string) [3]int {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	var out [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		n, _ := strconv.Atoi(parts[i])
		out[i] = n
	}
	return out
}

// showUpdateNotification checks whether the application was just updated and,
// if so, publishes a one-time AppUpdated fact (shown in the log panel) and
// clears the pending version.
func (u *updater) showUpdateNotification() {
	cfg := loadConfig()
	raw := strings.TrimSpace(cfg.UpdatedToVersion)
	if raw == "" {
		return
	}
	cfg.UpdatedToVersion = ""
	saveConfig(cfg)

	version := strings.TrimPrefix(raw, "v")
	if version != "" {
		u.bus.Publish(AppUpdated{Version: version})
	}
}

// restartSelf launches a fresh copy of exe with the original arguments then
// exits. The new process takes over; this one terminates cleanly.
func restartSelf(exe, version string) {
	setPendingUpdateVersion(version)

	args := os.Args[1:]
	cmd := exec.Command(exe, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Printf("updater: restart failed: %v — please restart manually", err)
		clearPendingUpdateVersion()
		return
	}
	// Give the new process a moment to start before we exit.
	time.Sleep(500 * time.Millisecond)
	os.Exit(0)
}
