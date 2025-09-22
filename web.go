package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// RingStatus holds the state of the last completed token. It is safe for concurrent use.
type RingStatus struct {
	mu            sync.RWMutex
	lastCompleted *Token
}

// Set stores the latest completed token.
func (s *RingStatus) Set(token *Token) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Store a copy of the token
	tokenCopy := *token
	s.lastCompleted = &tokenCopy
}

// Get retrieves a copy of the last completed token.
func (s *RingStatus) Get() *Token {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.lastCompleted == nil {
		return nil
	}
	// Return a copy to prevent race conditions
	tokenCopy := *s.lastCompleted
	return &tokenCopy
}

// WebAPI provides the /status endpoint.
type WebAPI struct {
	status *RingStatus
}

func NewWebAPI(status *RingStatus) *WebAPI {
	return &WebAPI{status: status}
}

// StatusResponse defines the JSON structure for the /status response.
type StatusResponse struct {
	Status          string    `json:"status"`
	LastCompletedAt *time.Time `json:"last_completed_at,omitempty"`
	Signers         []string  `json:"signers,omitempty"`
	Message         string    `json:"message,omitempty"`
}

// RegisterHandler sets up the HTTP handler for the API.
func (api *WebAPI) RegisterHandler() {
	http.HandleFunc("/status", api.handleStatus)
}

func (api *WebAPI) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	lastToken := api.status.Get()

	if lastToken == nil {
		json.NewEncoder(w).Encode(StatusResponse{
			Status:  "pending",
			Message: "No token has completed a full circle yet.",
		})
		return
	}

	completedAt := time.Unix(lastToken.IssuedAt, 0)
	json.NewEncoder(w).Encode(StatusResponse{
		Status:          "ok",
		LastCompletedAt: &completedAt,
		Signers:         lastToken.Signers,
	})
}
