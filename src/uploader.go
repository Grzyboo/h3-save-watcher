package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	apiBase    = "https://h3.fewbits.pl/api"
	headerSize = 16 * 1024
)

func (a *App) uploadFile(path string, fileType string) {
	a.uploadMu.Lock()
	a.uploadCount++
	a.uploadMu.Unlock()
	defer func() {
		a.uploadMu.Lock()
		a.uploadCount--
		if a.uploadCount < 0 {
			a.uploadCount = 0
		}
		a.uploadCond.Broadcast()
		a.uploadMu.Unlock()
	}()

	data, err := os.ReadFile(path)
	if err != nil {
		a.addLog(false, KeyLogReadError, a.relPath(path), err)
		return
	}

	filename := filepath.Base(path)
	hash := fmt.Sprintf("%x", sha256.Sum256(data))

	a.mu.Lock()
	if a.lastUploadedHash[fileType] == hash {
		a.mu.Unlock()
		log.Printf("skipping duplicate upload: %s (%s), hash %s already uploaded", filename, fileType, hash)
		return
	}
	a.lastUploadedHash[fileType] = hash
	info := a.gameInfo
	a.mu.Unlock()

	xHost := "0"
	if info.IsHost {
		xHost = "1"
	}
	gameHeaders := map[string]string{
		"X-Host":         xHost,
		"X-PlayerName":   info.PlayerName,
		"X-OpponentName": info.OpponentName,
		"X-GamePassword": info.Password,
		"X-AppVersion":   appVersion,
	}

	header := data
	if len(header) > headerSize {
		header = header[:headerSize]
	}
	headerRaw := base64.StdEncoding.EncodeToString(header)

	analyzeBody, _ := json.Marshal(map[string]any{
		"files": []map[string]any{{
			"filename":  filename,
			"hash":      hash,
			"rawSize":   len(data),
			"headerRaw": headerRaw,
		}},
	})

	analyzeResp, err := doRequest("POST", apiBase+"/upload/locked", "application/json", analyzeBody, a.instanceID, gameHeaders)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			a.addLog(false, KeyLogConnectionRefused, a.relPath(path))
		} else {
			a.addLog(false, KeyLogUploadError, a.relPath(path), err)
		}
		return
	}

	var analyzeResult struct {
		Results []struct {
			Filename      string `json:"filename"`
			Error         string `json:"error"`
			FileUploadKey string `json:"fileUploadKey"`
		} `json:"results"`
	}
	if err := json.Unmarshal(analyzeResp, &analyzeResult); err != nil {
		a.addLog(false, KeyLogInvalidAnalyzeResp, a.relPath(path))
		return
	}
	if len(analyzeResult.Results) == 0 {
		a.addLog(false, KeyLogEmptyResults, a.relPath(path))
		return
	}
	result := analyzeResult.Results[0]
	if result.Error != "" {
		if result.Error == "already_exists" {
			a.markFileAsSent(path, filename)
			log.Printf("upload %s: file already exists on server, cached as sent", filename)
			return
		}
		a.addLog(false, KeyLogServerError, a.relPath(path), result.Error)
		return
	}

	uploadResp, err := doRequest("POST", apiBase+"/upload/"+result.FileUploadKey, "application/octet-stream", data, a.instanceID, gameHeaders)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			a.addLog(false, KeyLogConnectionRefused, a.relPath(path))
		} else {
			a.addLog(false, KeyLogUploadError, a.relPath(path), err)
		}
		return
	}

	var uploadResult struct {
		Ok    bool   `json:"ok"`
		Path  string `json:"path"`
		UUID  string `json:"uuid"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(uploadResp, &uploadResult); err != nil {
		a.addLog(false, KeyLogInvalidUploadResp, a.relPath(path))
		return
	}
	if !uploadResult.Ok {
		if uploadResult.Error == "already_exists" {
			a.markFileAsSent(path, filename)
			log.Printf("upload %s: file already exists on server, cached as sent", filename)
			return
		}
		a.addLog(false, KeyLogServerRejected, a.relPath(path))
		return
	}

	a.addLog(true, KeyLogUploaded, a.relPath(path))

	a.sentFoldersMu.Lock()
	folder := a.watchedGameFolder
	if folder != "" {
		absPath, _ := filepath.Abs(path)
		absFolder, _ := filepath.Abs(folder)
		if strings.HasPrefix(absPath, absFolder+string(os.PathSeparator)) {
			a.sentFoldersCache.addFile(absFolder, filename)
			if err := a.sentFoldersCache.save(); err != nil {
				log.Printf("failed to save sent folders cache: %v", err)
			}
		}
	}
	a.sentFoldersMu.Unlock()
}

// markFileAsSent records a file in the sent-folders cache if it belongs to the
// currently watched game folder. This is used when the server reports that the
// file already exists, so the app does not keep retrying the upload.
func (a *App) markFileAsSent(path, filename string) {
	a.mu.Lock()
	folder := a.watchedGameFolder
	a.mu.Unlock()
	if folder == "" {
		return
	}

	absPath, _ := filepath.Abs(path)
	absFolder, _ := filepath.Abs(folder)
	if absFolder == "" || !strings.HasPrefix(absPath, absFolder+string(os.PathSeparator)) {
		return
	}

	a.sentFoldersMu.Lock()
	a.sentFoldersCache.addFile(absFolder, filename)
	if err := a.sentFoldersCache.save(); err != nil {
		log.Printf("failed to save sent folders cache: %v", err)
	}
	a.sentFoldersMu.Unlock()
}

func doRequest(method, url, contentType string, body []byte, instanceID string, extraHeaders map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(apiUser, apiPassword)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-InstanceId", instanceID)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
