package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
)

const (
	apiBase    = "https://h3.fewbits.pl/api"
	headerSize = 16 * 1024
)

func (a *App) uploadFile(path string, fileType string) {
	data, err := os.ReadFile(path)
	if err != nil {
		a.addLog(false, fmt.Sprintf(a.T(KeyLogReadError), a.relPath(path), err))
		return
	}

	filename := filepath.Base(path)
	hash := fmt.Sprintf("%x", sha256.Sum256(data))

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

	analyzeResp, err := doRequest("POST", apiBase+"/upload/locked", "application/json", analyzeBody, a.instanceID)
	if err != nil {
		msg := fmt.Sprintf(a.T(KeyLogUploadError), a.relPath(path), err)
		if errors.Is(err, syscall.ECONNREFUSED) {
			msg = fmt.Sprintf(a.T(KeyLogConnectionRefused), a.relPath(path))
		}
		a.addLog(false, msg)
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
		a.addLog(false, fmt.Sprintf(a.T(KeyLogInvalidAnalyzeResp), a.relPath(path)))
		return
	}
	if len(analyzeResult.Results) == 0 {
		a.addLog(false, fmt.Sprintf(a.T(KeyLogEmptyResults), a.relPath(path)))
		return
	}
	result := analyzeResult.Results[0]
	if result.Error != "" {
		a.addLog(false, fmt.Sprintf(a.T(KeyLogServerError), a.relPath(path), result.Error))
		return
	}

	uploadResp, err := doRequest("POST", apiBase+"/upload/"+result.FileUploadKey, "application/octet-stream", data, a.instanceID)
	if err != nil {
		msg := fmt.Sprintf(a.T(KeyLogUploadError), a.relPath(path), err)
		if errors.Is(err, syscall.ECONNREFUSED) {
			msg = fmt.Sprintf(a.T(KeyLogConnectionRefused), a.relPath(path))
		}
		a.addLog(false, msg)
		return
	}

	var uploadResult struct {
		Ok   bool   `json:"ok"`
		Path string `json:"path"`
		UUID string `json:"uuid"`
	}
	if err := json.Unmarshal(uploadResp, &uploadResult); err != nil {
		a.addLog(false, fmt.Sprintf(a.T(KeyLogInvalidUploadResp), a.relPath(path)))
		return
	}
	if !uploadResult.Ok {
		a.addLog(false, fmt.Sprintf(a.T(KeyLogServerRejected), a.relPath(path)))
		return
	}

	a.addLog(true, fmt.Sprintf(a.T(KeyLogUploaded), a.relPath(path)))
}

func doRequest(method, url, contentType string, body []byte, instanceID string) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(apiUser, apiPassword)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-InstanceId", instanceID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
