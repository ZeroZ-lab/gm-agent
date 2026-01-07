package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// HTTPConfig defines the HTTP server settings.
type HTTPConfig struct {
	Enable bool   `yaml:"enable" envconfig:"HTTP_ENABLE"`
	Addr   string `yaml:"addr" envconfig:"HTTP_ADDR"`
	APIKey string `yaml:"api_key" envconfig:"HTTP_API_KEY"`
}

// RuntimeRunner is the minimal runtime contract the API server relies on.
type RuntimeRunner interface {
	Ingest(ctx context.Context, event types.Event) error
	Run(ctx context.Context) error
}

// SessionResources contains runtime dependencies for a session.
type SessionResources struct {
	Runtime RuntimeRunner
	Store   store.Store
	Ctx     context.Context
	Cancel  context.CancelFunc
}

// SessionFactory creates per-session runtime resources.
type SessionFactory func(sessionID string) (*SessionResources, error)

type sessionRecord struct {
	id         string
	resources  *SessionResources
	status     string
	createdAt  time.Time
	lastError  string
	statusLock sync.Mutex
}

// Server hosts the Gin engine and session registry.
type Server struct {
	Engine   *gin.Engine
	sessions map[string]*sessionRecord
	factory  SessionFactory
	apiKey   string
	log      *slog.Logger
	ctx      context.Context
	mu       sync.RWMutex
}

// NewServer constructs the HTTP API server using the provided factory.
func NewServer(ctx context.Context, cfg HTTPConfig, factory SessionFactory, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}

	engine := gin.New()
	engine.Use(gin.Recovery())

	srv := &Server{
		Engine:   engine,
		sessions: make(map[string]*sessionRecord),
		factory:  factory,
		apiKey:   cfg.APIKey,
		log:      logger,
		ctx:      ctx,
	}

	engine.Use(srv.apiKeyMiddleware())

	v1 := engine.Group("/api/v1")
	v1.POST("/sessions", srv.handleCreateSession)
	v1.GET("/sessions/:id", srv.handleGetSession)
	v1.GET("/sessions/:id/events", srv.handleGetEvents)
	v1.POST("/sessions/:id/cancel", srv.handleCancel)

	engine.GET("/api/openapi.json", srv.handleOpenAPI)
	engine.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	return srv
}

func (s *Server) apiKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.apiKey == "" {
			return
		}
		key := c.GetHeader("X-API-Key")
		if key == "" || key != s.apiKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}
	}
}

func (s *Server) handleCreateSession(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
		return
	}

	sessionID := types.GenerateID("ses")
	resources, err := s.factory(sessionID)
	if err != nil {
		s.log.Error("failed to create session resources", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start session"})
		return
	}

	sess := &sessionRecord{
		id:        sessionID,
		resources: resources,
		status:    "running",
		createdAt: time.Now(),
	}

	s.mu.Lock()
	s.sessions[sessionID] = sess
	s.mu.Unlock()

	// Seed runtime with initial prompt
	event := &types.UserMessageEvent{
		BaseEvent: types.NewBaseEvent("user_request", "user", sessionID),
		Content:   req.Prompt,
		Priority:  10,
	}
	if err := sess.resources.Runtime.Ingest(sess.resources.Ctx, event); err != nil {
		s.log.Error("failed to ingest prompt", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ingest prompt"})
		return
	}

	go s.runSession(sess)

	c.JSON(http.StatusCreated, gin.H{
		"session_id": sessionID,
		"status":     sess.status,
		"created_at": sess.createdAt,
	})
}

func (s *Server) handleGetSession(c *gin.Context) {
	id := c.Param("id")
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	state, err := sess.resources.Store.LoadLatestState(c.Request.Context())
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load state"})
		return
	}

	sess.statusLock.Lock()
	status := sess.status
	errMsg := sess.lastError
	createdAt := sess.createdAt
	sess.statusLock.Unlock()

	resp := gin.H{
		"session_id": id,
		"status":     status,
		"created_at": createdAt,
	}
	if state != nil {
		resp["state"] = state
	}
	if errMsg != "" {
		resp["error"] = errMsg
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleGetEvents(c *gin.Context) {
	id := c.Param("id")
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	after := c.Query("after")
	events, err := sess.resources.Store.GetEventsSince(c.Request.Context(), after)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (s *Server) handleCancel(c *gin.Context) {
	id := c.Param("id")
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	sess.resources.Cancel()
	sess.statusLock.Lock()
	sess.status = "cancelled"
	sess.statusLock.Unlock()

	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

func (s *Server) handleOpenAPI(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write([]byte(openAPISchema))
}

func (s *Server) runSession(sess *sessionRecord) {
	defer sess.resources.Store.Close()

	err := sess.resources.Runtime.Run(sess.resources.Ctx)
	sess.statusLock.Lock()
	defer sess.statusLock.Unlock()

	if err != nil {
		if errors.Is(err, context.Canceled) {
			sess.status = "cancelled"
			return
		}
		sess.status = "error"
		sess.lastError = err.Error()
		return
	}
	sess.status = "completed"
}

const openAPISchema = `{
  "openapi": "3.0.0",
  "info": {
    "title": "gm-agent API",
    "version": "1.0.0"
  },
  "paths": {
    "/api/v1/sessions": {
      "post": {
        "summary": "Create a new agent session",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["prompt"],
                "properties": {
                  "prompt": {"type": "string"}
                }
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "session created",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "properties": {
                    "session_id": {"type": "string"},
                    "status": {"type": "string"}
                  }
                }
              }
            }
          }
        }
      }
    },
    "/api/v1/sessions/{id}": {
      "get": {
        "summary": "Get session status",
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": {"type": "string"}
          }
        ],
        "responses": {
          "200": {
            "description": "session status",
            "content": {
              "application/json": {
                "schema": {"type": "object"}
              }
            }
          },
          "404": {"description": "not found"}
        }
      }
    },
    "/api/v1/sessions/{id}/events": {
      "get": {
        "summary": "List events in a session",
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "string"}},
          {"name": "after", "in": "query", "schema": {"type": "string"}}
        ],
        "responses": {
          "200": {
            "description": "event list",
            "content": {"application/json": {"schema": {"type": "object"}}}
          }
        }
      }
    },
    "/api/v1/sessions/{id}/cancel": {
      "post": {
        "summary": "Cancel a running session",
        "parameters": [
          {"name": "id", "in": "path", "required": true, "schema": {"type": "string"}}
        ],
        "responses": {
          "200": {"description": "session cancelled"}
        }
      }
    }
  }
}`
