package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// clientErrorHTTPClient is used only for error reporting; the short timeout
// keeps the fire-and-forget calls lightweight.
var clientErrorHTTPClient = &http.Client{Timeout: 2 * time.Second}

// clientErrorPayload is the body of POST /api/client-errors. Name and
// InstanceID are nil (JSON null) when not established yet (e.g. an error
// while reading passwords.txt).
type clientErrorPayload struct {
	Message    string            `json:"message"`
	Name       *string           `json:"name"`
	InstanceID *string           `json:"instance_id"`
	Metadata   map[string]string `json:"metadata"`
}

// registerErrorHandlers forwards every red log entry to the server. The calls
// are lightweight and silent: no retries, failures are only written to the
// debug log.
func registerErrorHandlers(bus *Bus, s *State) {
	Subscribe(bus, func(e ClientErrorLogged) {
		// The message exactly as shown to the user, translated and formatted
		// in the current language (same formatting as LogEntry.Format).
		message := s.T(e.Key)
		if len(e.Args) > 0 {
			message = fmt.Sprintf(message, e.Args...)
		}
		go postClientError(message, string(e.Key), s.gameInfo.PlayerName, s.instanceID)
	})
}

// postClientError sends one error report to the server. Fire-and-forget: any
// failure is ignored.
func postClientError(message, code, name, instanceID string) {
	payload := clientErrorPayload{
		Message:  message,
		Metadata: map[string]string{"code": code},
	}
	if name != "" {
		payload.Name = &name
	}
	if instanceID != "" {
		payload.InstanceID = &instanceID
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", apiBase+"/client-errors", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.SetBasicAuth(apiUser, apiPassword)
	req.Header.Set("Content-Type", "application/json")

	resp, err := clientErrorHTTPClient.Do(req)
	if err != nil {
		log.Printf("client error report failed: %v", err)
		return
	}
	defer resp.Body.Close()
}
