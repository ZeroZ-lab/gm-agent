// @title           GM-Agent API
// @version         1.0
// @description     Agent runtime API for managing sessions.
// @host            localhost:8080
// @BasePath        /

package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gm-agent-org/gm-agent/pkg/api/middleware"
	"github.com/gm-agent-org/gm-agent/pkg/api/service"
)

// Config defines the HTTP server settings.
type Config struct {
	Enable  bool   `yaml:"enable" envconfig:"HTTP_ENABLE"`
	Addr    string `yaml:"addr" envconfig:"HTTP_ADDR"`
	APIKey  string `yaml:"api_key" envconfig:"HTTP_API_KEY"`
	DevMode bool   // Enables Swagger UI
}

// Server hosts the Gin engine and manages API resources.
type Server struct {
	engine     *gin.Engine
	config     Config
	sessionSvc *service.SessionService
	log        *slog.Logger
}

// NewServer constructs the HTTP API server.
func NewServer(cfg Config, sessionSvc *service.SessionService, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}

	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.Logger(log))

	srv := &Server{
		engine:     engine,
		config:     cfg,
		sessionSvc: sessionSvc,
		log:        log,
	}

	srv.setupRoutes()

	return srv
}

// Engine returns the underlying Gin engine (for http.Server).
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Addr returns the configured address.
func (s *Server) Addr() string {
	return s.config.Addr
}

// Run starts the HTTP server on the configured address.
func (s *Server) Run() error {
	s.log.Info("http api listening", "addr", s.config.Addr)
	return http.ListenAndServe(s.config.Addr, s.engine)
}
