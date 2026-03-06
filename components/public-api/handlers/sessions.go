package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"ambient-code-public-api/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ListSessions handles GET /v1/sessions
func ListSessions(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions", project)

	resp, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Forward non-OK responses with consistent error format
	if resp.StatusCode != http.StatusOK {
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	var backendResp struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(body, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Transform to simplified DTOs
	sessions := make([]types.SessionResponse, 0, len(backendResp.Items))
	for _, item := range backendResp.Items {
		sessions = append(sessions, transformSession(item))
	}

	c.JSON(http.StatusOK, types.SessionListResponse{
		Items: sessions,
		Total: len(sessions),
	})
}

// GetSession handles GET /v1/sessions/:id
func GetSession(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s", project, sessionID)

	resp, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		log.Printf("Backend request failed for session %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	var backendResp map[string]interface{}
	if err := json.Unmarshal(body, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, transformSession(backendResp))
}

// CreateSession handles POST /v1/sessions
func CreateSession(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}

	var req types.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Transform to backend format
	backendReq := map[string]interface{}{
		"initialPrompt": req.Task,
	}
	if req.DisplayName != "" {
		backendReq["displayName"] = req.DisplayName
	}
	if req.Model != "" {
		backendReq["model"] = req.Model
	}
	if len(req.Repos) > 0 {
		repos := make([]map[string]interface{}, len(req.Repos))
		for i, r := range req.Repos {
			repo := map[string]interface{}{
				"url": r.URL,
			}
			if r.Branch != "" {
				repo["branch"] = r.Branch
			}
			repos[i] = repo
		}
		backendReq["repos"] = repos
	}

	reqBody, err := json.Marshal(backendReq)
	if err != nil {
		log.Printf("Failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions", project)

	resp, err := ProxyRequest(c, http.MethodPost, path, reqBody)
	if err != nil {
		log.Printf("Backend request failed for create session: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		forwardErrorResponse(c, resp.StatusCode, respBody)
		return
	}

	// Parse response to get session ID
	var backendResp map[string]interface{}
	if err := json.Unmarshal(respBody, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	// Return simplified response
	c.JSON(http.StatusCreated, gin.H{
		"id":      backendResp["name"],
		"message": "Session created",
	})
}

// DeleteSession handles DELETE /v1/sessions/:id
func DeleteSession(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s", project, sessionID)

	resp, err := ProxyRequest(c, http.MethodDelete, path, nil)
	if err != nil {
		log.Printf("Backend request failed for delete session %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		c.Status(http.StatusNoContent)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	forwardErrorResponse(c, resp.StatusCode, body)
}

// SendMessage handles POST /v1/sessions/:id/message
func SendMessage(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	var req types.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build AG-UI RunAgentInput
	runID := uuid.NewString()
	aguiReq := map[string]interface{}{
		"runId":    runID,
		"threadId": sessionID,
		"messages": []map[string]interface{}{
			{"id": uuid.NewString(), "role": "user", "content": req.Content},
		},
	}

	reqBody, err := json.Marshal(aguiReq)
	if err != nil {
		log.Printf("Failed to marshal AG-UI request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/agui/run", project, sessionID)

	resp, err := ProxyRequest(c, http.MethodPost, path, reqBody)
	if err != nil {
		log.Printf("Backend request failed for send message to session %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		forwardErrorResponse(c, resp.StatusCode, respBody)
		return
	}

	c.JSON(http.StatusAccepted, types.SendMessageResponse{
		RunID:    runID,
		ThreadID: sessionID,
	})
}

// GetSessionOutput handles GET /v1/sessions/:id/output
// Supports optional ?run_id=<uuid> query param to filter events by run.
func GetSessionOutput(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	events, err := fetchSessionEvents(c, project, sessionID)
	if err != nil {
		return // fetchSessionEvents already wrote the error response
	}

	// If run_id query param is present, filter events to that run
	runID := c.Query("run_id")
	if runID != "" {
		if _, uuidErr := uuid.Parse(runID); uuidErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid run_id: must be a valid UUID"})
			return
		}

		filtered := filterEventsByRunID(events, runID)
		if len(filtered) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Run not found"})
			return
		}

		filteredJSON, marshalErr := json.Marshal(filtered)
		if marshalErr != nil {
			log.Printf("Failed to marshal filtered events: %v", marshalErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, types.SessionOutputResponse{
			SessionID: sessionID,
			Events:    json.RawMessage(filteredJSON),
		})
		return
	}

	// No run_id filter — return all events
	allEventsJSON, marshalErr := json.Marshal(events)
	if marshalErr != nil {
		log.Printf("Failed to marshal events: %v", marshalErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, types.SessionOutputResponse{
		SessionID: sessionID,
		Events:    json.RawMessage(allEventsJSON),
	})
}

// GetSessionRuns handles GET /v1/sessions/:id/runs
// Returns a summary of all AG-UI runs in the session.
func GetSessionRuns(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	events, err := fetchSessionEvents(c, project, sessionID)
	if err != nil {
		return // fetchSessionEvents already wrote the error response
	}

	runs := buildRunSummaries(events)

	c.JSON(http.StatusOK, types.SessionRunsResponse{
		SessionID: sessionID,
		Runs:      runs,
	})
}

// StartSession handles POST /v1/sessions/:id/start
func StartSession(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/start", project, sessionID)

	resp, err := ProxyRequest(c, http.MethodPost, path, nil)
	if err != nil {
		log.Printf("Backend request failed for start session %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	var backendResp map[string]interface{}
	if err := json.Unmarshal(body, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusAccepted, transformSession(backendResp))
}

// StopSession handles POST /v1/sessions/:id/stop
func StopSession(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/stop", project, sessionID)

	resp, err := ProxyRequest(c, http.MethodPost, path, nil)
	if err != nil {
		log.Printf("Backend request failed for stop session %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	var backendResp map[string]interface{}
	if err := json.Unmarshal(body, &backendResp); err != nil {
		log.Printf("Failed to parse backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusAccepted, transformSession(backendResp))
}

// InterruptSession handles POST /v1/sessions/:id/interrupt
func InterruptSession(c *gin.Context) {
	project := GetProject(c)
	if !ValidateProjectName(project) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project name"})
		return
	}
	sessionID := c.Param("id")
	if !ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/agui/interrupt", project, sessionID)

	resp, err := ProxyRequest(c, http.MethodPost, path, []byte("{}"))
	if err != nil {
		log.Printf("Backend request failed for interrupt session %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		forwardErrorResponse(c, resp.StatusCode, body)
		return
	}

	c.JSON(http.StatusOK, types.MessageResponse{Message: "Interrupt signal sent"})
}

// fetchSessionEvents calls the backend export endpoint and returns parsed AG-UI events.
// On error, it writes the HTTP error response to c and returns a non-nil error.
func fetchSessionEvents(c *gin.Context, project, sessionID string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/export", project, sessionID)

	resp, err := ProxyRequest(c, http.MethodGet, path, nil)
	if err != nil {
		log.Printf("Backend request failed for session output %s: %v", sessionID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Backend unavailable"})
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read backend response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		forwardErrorResponse(c, resp.StatusCode, body)
		return nil, fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	// Parse backend ExportResponse to extract aguiEvents
	var backendResp map[string]json.RawMessage
	if err := json.Unmarshal(body, &backendResp); err != nil {
		log.Printf("Failed to parse backend export response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return nil, err
	}

	eventsRaw := json.RawMessage("[]")
	if aguiEvents, ok := backendResp["aguiEvents"]; ok {
		eventsRaw = aguiEvents
	}

	var events []map[string]interface{}
	if err := json.Unmarshal(eventsRaw, &events); err != nil {
		log.Printf("Failed to parse AG-UI events: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return nil, err
	}

	return events, nil
}

// filterEventsByRunID returns only events whose "runId" field matches the given run ID.
func filterEventsByRunID(events []map[string]interface{}, runID string) []map[string]interface{} {
	var filtered []map[string]interface{}
	for _, event := range events {
		if eventRunID, ok := event["runId"].(string); ok && eventRunID == runID {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// buildRunSummaries scans AG-UI events and builds a summary for each run.
func buildRunSummaries(events []map[string]interface{}) []types.RunSummary {
	type runState struct {
		summary types.RunSummary
		order   int
	}

	runs := make(map[string]*runState)
	orderCounter := 0

	for _, event := range events {
		runID, ok := event["runId"].(string)
		if !ok || runID == "" {
			continue
		}

		state, exists := runs[runID]
		if !exists {
			state = &runState{
				summary: types.RunSummary{
					RunID:  runID,
					Status: "running",
				},
				order: orderCounter,
			}
			orderCounter++
			runs[runID] = state
		}
		state.summary.EventCount++

		eventType, _ := event["type"].(string)
		switch eventType {
		case "RUN_STARTED":
			if ts, ok := toInt64(event["timestamp"]); ok {
				state.summary.StartedAt = ts
			}
		case "RUN_FINISHED":
			state.summary.Status = "completed"
			if ts, ok := toInt64(event["timestamp"]); ok {
				state.summary.FinishedAt = ts
			}
		case "RUN_ERROR":
			state.summary.Status = "error"
			if ts, ok := toInt64(event["timestamp"]); ok {
				state.summary.FinishedAt = ts
			}
		case "TEXT_MESSAGE_START":
			// Capture the user message that started this run
			if role, ok := event["role"].(string); ok && role == "user" {
				if content, ok := event["content"].(string); ok && state.summary.UserMessage == "" {
					state.summary.UserMessage = content
				}
			}
		}
	}

	// Sort by order of first appearance
	result := make([]types.RunSummary, 0, len(runs))
	ordered := make([]*runState, 0, len(runs))
	for _, state := range runs {
		ordered = append(ordered, state)
	}
	for i := 0; i < len(ordered)-1; i++ {
		for j := i + 1; j < len(ordered); j++ {
			if ordered[j].order < ordered[i].order {
				ordered[i], ordered[j] = ordered[j], ordered[i]
			}
		}
	}
	for _, state := range ordered {
		result = append(result, state.summary)
	}

	return result
}

// toInt64 attempts to extract an int64 from an interface value (handles JSON number types).
func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}

// forwardErrorResponse forwards backend error with consistent JSON format
func forwardErrorResponse(c *gin.Context, statusCode int, body []byte) {
	// Try to parse as JSON error response
	var errorResp map[string]interface{}
	if err := json.Unmarshal(body, &errorResp); err == nil {
		// Backend returned valid JSON, forward it
		c.JSON(statusCode, errorResp)
		return
	}

	// Backend returned non-JSON, wrap in standard error format
	c.JSON(statusCode, gin.H{"error": "Request failed"})
}

// transformSession converts backend session format to simplified DTO
func transformSession(data map[string]interface{}) types.SessionResponse {
	session := types.SessionResponse{}

	// Extract metadata
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		if name, ok := metadata["name"].(string); ok {
			session.ID = name
		}
		if creationTimestamp, ok := metadata["creationTimestamp"].(string); ok {
			session.CreatedAt = creationTimestamp
		}
	}

	// If no metadata, try top-level name (list response format)
	if session.ID == "" {
		if name, ok := data["name"].(string); ok {
			session.ID = name
		}
	}

	// Extract spec
	if spec, ok := data["spec"].(map[string]interface{}); ok {
		if prompt, ok := spec["initialPrompt"].(string); ok {
			session.Task = prompt
		}
		if displayName, ok := spec["displayName"].(string); ok {
			session.DisplayName = displayName
		}
		if model, ok := spec["model"].(string); ok {
			session.Model = model
		}
	}

	// Extract status
	if status, ok := data["status"].(map[string]interface{}); ok {
		if phase, ok := status["phase"].(string); ok {
			session.Status = normalizePhase(phase)
		}
		if completionTime, ok := status["completionTime"].(string); ok {
			session.CompletedAt = completionTime
		}
		if result, ok := status["result"].(string); ok {
			session.Result = result
		}
		if errMsg, ok := status["error"].(string); ok {
			session.Error = errMsg
		}
	}

	// Default status if not set
	if session.Status == "" {
		session.Status = "pending"
	}

	return session
}

// normalizePhase converts K8s phase to simplified status
func normalizePhase(phase string) string {
	switch phase {
	case "Pending", "Creating", "Initializing":
		return "pending"
	case "Running", "Active":
		return "running"
	case "Completed", "Succeeded":
		return "completed"
	case "Failed", "Error":
		return "failed"
	default:
		return phase
	}
}
