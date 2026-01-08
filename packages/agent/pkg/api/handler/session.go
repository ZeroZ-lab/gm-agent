package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gm-agent-org/gm-agent/pkg/api/dto"
	"github.com/gm-agent-org/gm-agent/pkg/api/service"
)

// SessionHandler handles session-related requests.
type SessionHandler struct {
	svc *service.SessionService
}

// NewSessionHandler creates a new SessionHandler.
func NewSessionHandler(svc *service.SessionService) *SessionHandler {
	return &SessionHandler{svc: svc}
}

// Create godoc
// @Summary      Create a new session
// @Description  Start a new agent session with the given prompt
// @Tags         session
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateSessionRequest true "Session request"
// @Success      201 {object} dto.SessionResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/session [post]
func (h *SessionHandler) Create(c *gin.Context) {
	var req dto.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body for session creation without prompt
		req = dto.CreateSessionRequest{}
	}

	session, err := h.svc.Create(c.Request.Context(), req.Prompt, req.SystemPrompt, req.Priority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.SessionResponse{
		ID:        session.ID,
		Status:    session.Status,
		CreatedAt: session.CreatedAt,
	})
}

// List godoc
// @Summary      List all sessions
// @Description  Returns a list of all sessions
// @Tags         session
// @Produce      json
// @Success      200 {object} dto.SessionListResponse
// @Router       /api/v1/session [get]
func (h *SessionHandler) List(c *gin.Context) {
	sessions := h.svc.List()

	resp := dto.SessionListResponse{
		Sessions: make([]dto.SessionResponse, 0, len(sessions)),
	}
	for _, sess := range sessions {
		status, lastErr := sess.GetStatus()
		resp.Sessions = append(resp.Sessions, dto.SessionResponse{
			ID:        sess.ID,
			Status:    status,
			CreatedAt: sess.CreatedAt,
			Error:     lastErr,
		})
	}

	c.JSON(http.StatusOK, resp)
}

// Get godoc
// @Summary      Get session details
// @Description  Retrieve the current status and state of a session
// @Tags         session
// @Produce      json
// @Param        id path string true "Session ID"
// @Success      200 {object} dto.SessionResponse
// @Failure      404 {object} dto.ErrorResponse
// @Router       /api/v1/session/{id} [get]
func (h *SessionHandler) Get(c *gin.Context) {
	id := c.Param("id")

	session, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	status, lastErr := session.GetStatus()
	c.JSON(http.StatusOK, dto.SessionResponse{
		ID:        session.ID,
		Status:    status,
		CreatedAt: session.CreatedAt,
		Error:     lastErr,
	})
}

// Delete godoc
// @Summary      Delete a session
// @Description  Delete a session and its resources
// @Tags         session
// @Produce      json
// @Param        id path string true "Session ID"
// @Success      200 {object} dto.DeleteResponse
// @Failure      404 {object} dto.ErrorResponse
// @Router       /api/v1/session/{id} [delete]
func (h *SessionHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.svc.Delete(id)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.DeleteResponse{Deleted: true})
}

// Cancel godoc
// @Summary      Cancel a session
// @Description  Cancel a running session
// @Tags         session
// @Produce      json
// @Param        id path string true "Session ID"
// @Success      200 {object} dto.SessionResponse
// @Failure      404 {object} dto.ErrorResponse
// @Router       /api/v1/session/{id}/cancel [post]
func (h *SessionHandler) Cancel(c *gin.Context) {
	id := c.Param("id")

	err := h.svc.Cancel(id)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.SessionResponse{
		ID:     id,
		Status: "cancelled",
	})
}

// Message godoc
// @Summary      Send a message to a session
// @Description  Send a new user message or command to an active session
// @Tags         session
// @Accept       json
// @Produce      json
// @Param        id path string true "Session ID"
// @Param        request body dto.MessageRequest true "Message request"
// @Success      200 {object} dto.SessionResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/session/{id}/message [post]
func (h *SessionHandler) Message(c *gin.Context) {
	id := c.Param("id")
	var req dto.MessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body"})
		return
	}

	session, err := h.svc.Message(c.Request.Context(), id, req.Content, req.Semantic)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	status, lastErr := session.GetStatus()
	c.JSON(http.StatusOK, dto.SessionResponse{
		ID:        session.ID,
		Status:    status,
		CreatedAt: session.CreatedAt,
		Error:     lastErr,
	})
}

// Permission godoc
// @Summary      Respond to permission request
// @Description  Approve or deny a permission request for a session
// @Tags         session
// @Accept       json
// @Produce      json
// @Param        id path string true "Session ID"
// @Param        request body dto.PermissionResponseRequest true "Permission response"
// @Success      200 {object} dto.SessionResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/session/{id}/permission [post]
func (h *SessionHandler) Permission(c *gin.Context) {
	id := c.Param("id")
	var req dto.PermissionResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body"})
		return
	}

	if err := h.svc.RespondPermission(id, req.RequestID, req.Approved, req.Always); err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.SessionResponse{ID: id, Status: "ok"})
}

// SSE godoc
// @Summary      SSE Event Stream
// @Description  Server-Sent Events stream for real-time session updates
// @Tags         session
// @Produce      text/event-stream
// @Param        id path string true "Session ID"
// @Param        after query string false "Event ID cursor"
// @Router       /api/v1/session/{id}/event [get]
func (h *SessionHandler) SSE(c *gin.Context) {
	id := c.Param("id")

	session, err := h.svc.Get(id)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// Get cursor
	afterEventID := c.Query("after")

	ctx := c.Request.Context()
	ticker := time.NewTicker(50 * time.Millisecond) // Fast polling for smooth streaming
	defer ticker.Stop()

	lastEventID := afterEventID

	// Send initial connected event
	_, _ = c.Writer.Write([]byte("event: connected\ndata: {\"session_id\":\"" + id + "\"}\n\n"))
	c.Writer.(http.Flusher).Flush()

	for {
		select {
		case <-ctx.Done():
			return
		case <-session.Resources.Ctx.Done():
			// Session ended
			status, _ := session.GetStatus()
			_, _ = c.Writer.Write([]byte("event: session_ended\ndata: {\"status\":\"" + status + "\"}\n\n"))
			c.Writer.(http.Flusher).Flush()
			return
		case <-ticker.C:
			// Poll for new events
			events, err := session.Resources.Store.GetEventsSince(ctx, lastEventID)
			if err != nil {
				continue
			}
			for _, evt := range events {
				evtID := evt.EventID()
				evtType := evt.EventType()

				data, err := json.Marshal(evt)
				if err != nil {
					continue
				}

				_, _ = c.Writer.Write([]byte("event: " + evtType + "\ndata: " + string(data) + "\n\n"))
				lastEventID = evtID
			}
			if len(events) > 0 {
				c.Writer.(http.Flusher).Flush()
			}
		}
	}
}

// ListCheckpoints godoc
// @Summary      List checkpoints
// @Description  Retrieve all checkpoints for a session
// @Tags         session
// @Produce      json
// @Param        id path string true "Session ID"
// @Success      200 {object} dto.CheckpointListResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/session/{id}/checkpoints [get]
func (h *SessionHandler) ListCheckpoints(c *gin.Context) {
	id := c.Param("id")

	checkpoints, err := h.svc.ListCheckpoints(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, checkpoints)
}

// Rewind godoc
// @Summary      Rewind session
// @Description  Rewind a session to a previous checkpoint
// @Tags         session
// @Accept       json
// @Produce      json
// @Param        id path string true "Session ID"
// @Param        request body dto.RewindRequest true "Rewind request"
// @Success      200 {object} dto.RewindResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/session/{id}/rewind [post]
func (h *SessionHandler) Rewind(c *gin.Context) {
	id := c.Param("id")
	var req dto.RewindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body"})
		return
	}

	result, err := h.svc.Rewind(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
