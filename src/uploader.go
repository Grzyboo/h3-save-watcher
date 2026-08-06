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
	"path/filepath"
	"syscall"
)

const (
	apiBase    = "https://h3.fewbits.pl/api"
	headerSize = 16 * 1024
)

type uploadResult int

const (
	uploadOK uploadResult = iota
	uploadAlreadyExists
	uploadError
)

// uploadOutcome is the result of one upload attempt; kind/err are set only
// when result is uploadError.
type uploadOutcome struct {
	result uploadResult
	kind   UploadErrorKind
	err    error
}

// uploadSaveFile uploads one save file to the server: analyze (locked) first,
// then the actual upload. It is pure — no App state, no logging; the outcome
// is reported back to the caller.
func uploadSaveFile(path string, data []byte, info GameInfo, instanceID string) uploadOutcome {
	filename := filepath.Base(path)
	hash := fmt.Sprintf("%x", sha256.Sum256(data))

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

	analyzeResp, err := doRequest("POST", apiBase+"/upload/locked", "application/json", analyzeBody, instanceID, gameHeaders)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			return uploadOutcome{result: uploadError, kind: UploadErrConnectionRefused, err: err}
		}
		return uploadOutcome{result: uploadError, kind: UploadErrRequest, err: err}
	}

	var analyzeResult struct {
		Results []struct {
			Filename      string `json:"filename"`
			Error         string `json:"error"`
			FileUploadKey string `json:"fileUploadKey"`
		} `json:"results"`
	}
	if err := json.Unmarshal(analyzeResp, &analyzeResult); err != nil {
		return uploadOutcome{result: uploadError, kind: UploadErrInvalidAnalyzeResp, err: err}
	}
	if len(analyzeResult.Results) == 0 {
		return uploadOutcome{result: uploadError, kind: UploadErrEmptyResults}
	}
	result := analyzeResult.Results[0]
	if result.Error != "" {
		if result.Error == "already_exists" {
			return uploadOutcome{result: uploadAlreadyExists}
		}
		return uploadOutcome{result: uploadError, kind: UploadErrServer, err: errors.New(result.Error)}
	}

	uploadResp, err := doRequest("POST", apiBase+"/upload/"+result.FileUploadKey, "application/octet-stream", data, instanceID, gameHeaders)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			return uploadOutcome{result: uploadError, kind: UploadErrConnectionRefused, err: err}
		}
		return uploadOutcome{result: uploadError, kind: UploadErrRequest, err: err}
	}

	var uploadResult struct {
		Ok    bool   `json:"ok"`
		Path  string `json:"path"`
		UUID  string `json:"uuid"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(uploadResp, &uploadResult); err != nil {
		return uploadOutcome{result: uploadError, kind: UploadErrInvalidUploadResp, err: err}
	}
	if !uploadResult.Ok {
		if uploadResult.Error == "already_exists" {
			return uploadOutcome{result: uploadAlreadyExists}
		}
		return uploadOutcome{result: uploadError, kind: UploadErrServerRejected}
	}

	return uploadOutcome{result: uploadOK}
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
