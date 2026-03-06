package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ambient-code-public-api/types"

	"github.com/gin-gonic/gin"
)

func TestTransformSession(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected types.SessionResponse
	}{
		{
			name: "Full session with metadata and status",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "session-123",
					"creationTimestamp": "2026-01-29T10:00:00Z",
				},
				"spec": map[string]interface{}{
					"initialPrompt": "Fix the bug",
					"model":  "claude-sonnet-4",
				},
				"status": map[string]interface{}{
					"phase":          "Running",
					"completionTime": "",
				},
			},
			expected: types.SessionResponse{
				ID:        "session-123",
				Status:    "running",
				Task:      "Fix the bug",
				Model:     "claude-sonnet-4",
				CreatedAt: "2026-01-29T10:00:00Z",
			},
		},
		{
			name: "Completed session with result",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "session-456",
					"creationTimestamp": "2026-01-29T09:00:00Z",
				},
				"spec": map[string]interface{}{
					"initialPrompt": "Refactor code",
				},
				"status": map[string]interface{}{
					"phase":          "Completed",
					"completionTime": "2026-01-29T09:30:00Z",
					"result":         "Successfully refactored",
				},
			},
			expected: types.SessionResponse{
				ID:          "session-456",
				Status:      "completed",
				Task:        "Refactor code",
				CreatedAt:   "2026-01-29T09:00:00Z",
				CompletedAt: "2026-01-29T09:30:00Z",
				Result:      "Successfully refactored",
			},
		},
		{
			name: "Failed session with error",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "session-789",
					"creationTimestamp": "2026-01-29T08:00:00Z",
				},
				"spec": map[string]interface{}{
					"initialPrompt": "Do something",
				},
				"status": map[string]interface{}{
					"phase": "Failed",
					"error": "Something went wrong",
				},
			},
			expected: types.SessionResponse{
				ID:        "session-789",
				Status:    "failed",
				Task:      "Do something",
				CreatedAt: "2026-01-29T08:00:00Z",
				Error:     "Something went wrong",
			},
		},
		{
			name: "List response format (name at top level)",
			input: map[string]interface{}{
				"name": "session-list-item",
				"spec": map[string]interface{}{
					"initialPrompt": "List item task",
				},
				"status": map[string]interface{}{
					"phase": "Pending",
				},
			},
			expected: types.SessionResponse{
				ID:     "session-list-item",
				Status: "pending",
				Task:   "List item task",
			},
		},
		{
			name:  "Empty session",
			input: map[string]interface{}{},
			expected: types.SessionResponse{
				Status: "pending",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformSession(tt.input)

			if result.ID != tt.expected.ID {
				t.Errorf("ID = %q, want %q", result.ID, tt.expected.ID)
			}
			if result.Status != tt.expected.Status {
				t.Errorf("Status = %q, want %q", result.Status, tt.expected.Status)
			}
			if result.Task != tt.expected.Task {
				t.Errorf("Task = %q, want %q", result.Task, tt.expected.Task)
			}
			if result.Model != tt.expected.Model {
				t.Errorf("Model = %q, want %q", result.Model, tt.expected.Model)
			}
			if result.CreatedAt != tt.expected.CreatedAt {
				t.Errorf("CreatedAt = %q, want %q", result.CreatedAt, tt.expected.CreatedAt)
			}
			if result.CompletedAt != tt.expected.CompletedAt {
				t.Errorf("CompletedAt = %q, want %q", result.CompletedAt, tt.expected.CompletedAt)
			}
			if result.Result != tt.expected.Result {
				t.Errorf("Result = %q, want %q", result.Result, tt.expected.Result)
			}
			if result.Error != tt.expected.Error {
				t.Errorf("Error = %q, want %q", result.Error, tt.expected.Error)
			}
		})
	}
}

func TestNormalizePhase(t *testing.T) {
	tests := []struct {
		phase    string
		expected string
	}{
		{"Pending", "pending"},
		{"Creating", "pending"},
		{"Initializing", "pending"},
		{"Running", "running"},
		{"Active", "running"},
		{"Completed", "completed"},
		{"Succeeded", "completed"},
		{"Failed", "failed"},
		{"Error", "failed"},
		{"Unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			result := normalizePhase(tt.phase)
			if result != tt.expected {
				t.Errorf("normalizePhase(%q) = %q, want %q", tt.phase, result, tt.expected)
			}
		})
	}
}

func TestForwardErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		statusCode     int
		body           []byte
		expectedStatus int
		expectJSON     bool
	}{
		{
			name:           "Backend returns JSON error",
			statusCode:     500,
			body:           []byte(`{"error": "Backend error message"}`),
			expectedStatus: 500,
			expectJSON:     true,
		},
		{
			name:           "Backend returns 404 JSON",
			statusCode:     404,
			body:           []byte(`{"error": "Session not found"}`),
			expectedStatus: 404,
			expectJSON:     true,
		},
		{
			name:           "Backend returns non-JSON (plain text)",
			statusCode:     502,
			body:           []byte("Bad Gateway"),
			expectedStatus: 502,
			expectJSON:     true, // Should be wrapped in JSON
		},
		{
			name:           "Backend returns malformed JSON",
			statusCode:     500,
			body:           []byte(`{"error": "incomplete`),
			expectedStatus: 500,
			expectJSON:     true, // Should be wrapped in generic JSON
		},
		{
			name:           "Backend returns empty body",
			statusCode:     500,
			body:           []byte{},
			expectedStatus: 500,
			expectJSON:     true, // Should be wrapped in generic JSON
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)

			forwardErrorResponse(c, tt.statusCode, tt.body)

			if w.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if tt.expectJSON {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json; charset=utf-8" {
					t.Errorf("Content-Type = %q, want application/json", contentType)
				}
			}
		})
	}
}

// setupTestBackend creates a test HTTP server that mimics the backend for session lifecycle tests.
// It returns the server and a cleanup function that restores BackendURL.
func setupTestBackend(t *testing.T, expectedPath, expectedMethod string, responseStatus int, responseBody interface{}) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Errorf("unexpected path: got %s, want %s", r.URL.Path, expectedPath)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != expectedMethod {
			t.Errorf("unexpected method: got %s, want %s", r.Method, expectedMethod)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Verify Authorization header is forwarded
		if auth := r.Header.Get("Authorization"); auth == "" {
			t.Error("expected Authorization header to be forwarded")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(responseStatus)
		json.NewEncoder(w).Encode(responseBody)
	}))
	return server
}

func TestE2E_StartSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backendResp := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":              "session-abc",
			"creationTimestamp": "2026-01-29T10:00:00Z",
		},
		"spec": map[string]interface{}{
			"initialPrompt": "Fix the bug",
		},
		"status": map[string]interface{}{
			"phase": "Running",
		},
	}

	server := setupTestBackend(t, "/api/projects/my-project/agentic-sessions/session-abc/start", "POST", http.StatusOK, backendResp)
	defer server.Close()
	originalURL := BackendURL
	BackendURL = server.URL
	defer func() { BackendURL = originalURL }()

	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.POST("/v1/sessions/:id/start", func(ctx *gin.Context) {
		ctx.Set("project", "my-project")
		ctx.Set("token", "test-token")
		StartSession(ctx)
	})
	c.Request = httptest.NewRequest("POST", "/v1/sessions/session-abc/start", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	engine.ServeHTTP(w, c.Request)

	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var resp types.SessionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.ID != "session-abc" {
		t.Errorf("ID = %q, want %q", resp.ID, "session-abc")
	}
	if resp.Status != "running" {
		t.Errorf("Status = %q, want %q", resp.Status, "running")
	}
}

func TestE2E_StopSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	backendResp := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":              "session-abc",
			"creationTimestamp": "2026-01-29T10:00:00Z",
		},
		"spec": map[string]interface{}{
			"initialPrompt": "Fix the bug",
		},
		"status": map[string]interface{}{
			"phase": "Completed",
		},
	}

	server := setupTestBackend(t, "/api/projects/my-project/agentic-sessions/session-abc/stop", "POST", http.StatusOK, backendResp)
	defer server.Close()
	originalURL := BackendURL
	BackendURL = server.URL
	defer func() { BackendURL = originalURL }()

	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.POST("/v1/sessions/:id/stop", func(ctx *gin.Context) {
		ctx.Set("project", "my-project")
		ctx.Set("token", "test-token")
		StopSession(ctx)
	})
	c.Request = httptest.NewRequest("POST", "/v1/sessions/session-abc/stop", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	engine.ServeHTTP(w, c.Request)

	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var resp types.SessionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.ID != "session-abc" {
		t.Errorf("ID = %q, want %q", resp.ID, "session-abc")
	}
	if resp.Status != "completed" {
		t.Errorf("Status = %q, want %q", resp.Status, "completed")
	}
}

func TestE2E_InterruptSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := setupTestBackend(t, "/api/projects/my-project/agentic-sessions/session-abc/agui/interrupt", "POST", http.StatusOK, map[string]string{"message": "ok"})
	defer server.Close()
	originalURL := BackendURL
	BackendURL = server.URL
	defer func() { BackendURL = originalURL }()

	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.POST("/v1/sessions/:id/interrupt", func(ctx *gin.Context) {
		ctx.Set("project", "my-project")
		ctx.Set("token", "test-token")
		InterruptSession(ctx)
	})
	c.Request = httptest.NewRequest("POST", "/v1/sessions/session-abc/interrupt", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	engine.ServeHTTP(w, c.Request)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp types.MessageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Message != "Interrupt signal sent" {
		t.Errorf("Message = %q, want %q", resp.Message, "Interrupt signal sent")
	}
}

func TestE2E_StartSession_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.POST("/v1/sessions/:id/start", func(ctx *gin.Context) {
		ctx.Set("project", "my-project")
		ctx.Set("token", "test-token")
		StartSession(ctx)
	})
	c.Request = httptest.NewRequest("POST", "/v1/sessions/INVALID_SESSION/start", nil)
	engine.ServeHTTP(w, c.Request)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestE2E_StopSession_BackendError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := setupTestBackend(t, "/api/projects/my-project/agentic-sessions/session-abc/stop", "POST", http.StatusNotFound, map[string]string{"error": "Session not found"})
	defer server.Close()
	originalURL := BackendURL
	BackendURL = server.URL
	defer func() { BackendURL = originalURL }()

	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.POST("/v1/sessions/:id/stop", func(ctx *gin.Context) {
		ctx.Set("project", "my-project")
		ctx.Set("token", "test-token")
		StopSession(ctx)
	})
	c.Request = httptest.NewRequest("POST", "/v1/sessions/session-abc/stop", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	engine.ServeHTTP(w, c.Request)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestE2E_InterruptSession_BackendError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := setupTestBackend(t, "/api/projects/my-project/agentic-sessions/session-abc/agui/interrupt", "POST", http.StatusConflict, map[string]string{"error": "Session not running"})
	defer server.Close()
	originalURL := BackendURL
	BackendURL = server.URL
	defer func() { BackendURL = originalURL }()

	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.POST("/v1/sessions/:id/interrupt", func(ctx *gin.Context) {
		ctx.Set("project", "my-project")
		ctx.Set("token", "test-token")
		InterruptSession(ctx)
	})
	c.Request = httptest.NewRequest("POST", "/v1/sessions/session-abc/interrupt", nil)
	c.Request.Header.Set("Authorization", "Bearer test-token")
	engine.ServeHTTP(w, c.Request)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestTransformSession_TypeSafety(t *testing.T) {
	// Test that transformSession handles incorrect types gracefully
	tests := []struct {
		name  string
		input map[string]interface{}
	}{
		{
			name: "Metadata is wrong type",
			input: map[string]interface{}{
				"metadata": "not-a-map",
			},
		},
		{
			name: "Spec is wrong type",
			input: map[string]interface{}{
				"spec": []string{"not", "a", "map"},
			},
		},
		{
			name: "Status is wrong type",
			input: map[string]interface{}{
				"status": 12345,
			},
		},
		{
			name: "Nested fields are wrong types",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              12345, // should be string
					"creationTimestamp": true,  // should be string
				},
				"spec": map[string]interface{}{
					"prompt": []byte("bytes"), // should be string
					"model":  nil,
				},
				"status": map[string]interface{}{
					"phase":  map[string]string{}, // should be string
					"result": 99.9,                // should be string
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := transformSession(tt.input)
			// Should return valid (though possibly empty) response
			if result.Status == "" {
				result.Status = "pending" // default is applied
			}
			// Just verify no panic occurred
		})
	}
}
