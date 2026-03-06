package types

import "encoding/json"

// SessionResponse is the simplified session response for the public API
type SessionResponse struct {
	ID          string `json:"id"`
	Status      string `json:"status"` // "pending", "running", "completed", "failed"
	DisplayName string `json:"display_name,omitempty"`
	Task        string `json:"task"`
	Model       string `json:"model,omitempty"`
	CreatedAt   string `json:"createdAt"`
	CompletedAt string `json:"completedAt,omitempty"`
	Result      string `json:"result,omitempty"`
	Error       string `json:"error,omitempty"`
}

// SessionListResponse is the response for listing sessions
type SessionListResponse struct {
	Items []SessionResponse `json:"items"`
	Total int               `json:"total"`
}

// CreateSessionRequest is the request body for creating a session
type CreateSessionRequest struct {
	Task        string `json:"task" binding:"required"`
	DisplayName string `json:"display_name,omitempty"`
	Model       string `json:"model,omitempty"`
	Repos       []Repo `json:"repos,omitempty"`
}

// Repo represents a repository configuration
type Repo struct {
	URL    string `json:"url" binding:"required"`
	Branch string `json:"branch,omitempty"`
}

// SendMessageRequest is the request body for sending a message to a session
type SendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// SendMessageResponse is the response after sending a message
type SendMessageResponse struct {
	RunID    string `json:"run_id"`
	ThreadID string `json:"thread_id"`
}

// SessionOutputResponse is the response for getting session output
type SessionOutputResponse struct {
	SessionID string          `json:"session_id"`
	Events    json.RawMessage `json:"events"`
}

// RunSummary represents a single AG-UI run within a session
type RunSummary struct {
	RunID       string `json:"run_id"`
	StartedAt   int64  `json:"started_at,omitempty"`
	FinishedAt  int64  `json:"finished_at,omitempty"`
	Status      string `json:"status"`
	UserMessage string `json:"user_message,omitempty"`
	EventCount  int    `json:"event_count"`
}

// SessionRunsResponse is the response for listing runs in a session
type SessionRunsResponse struct {
	SessionID string       `json:"session_id"`
	Runs      []RunSummary `json:"runs"`
}

// ErrorResponse is a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
