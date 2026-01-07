package dto

import "time"

// SessionResponse is the response for a single session.
type SessionResponse struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Error     string    `json:"error,omitempty"`
}

// SessionListResponse is the response for listing sessions.
type SessionListResponse struct {
	Sessions []SessionResponse `json:"sessions"`
}

// HealthResponse is the response for health check.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ErrorResponse is a standard error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// DeleteResponse is the response for delete operations.
type DeleteResponse struct {
	Deleted bool `json:"deleted"`
}
