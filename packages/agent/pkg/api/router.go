package api

import (
	"github.com/gm-agent-org/gm-agent/pkg/api/handler"
	"github.com/gm-agent-org/gm-agent/pkg/api/middleware"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// setupRoutes configures all API routes.
func (s *Server) setupRoutes() {
	// Health (no auth required)
	s.engine.GET("/health", handler.Health)

	// API v1 group
	v1 := s.engine.Group("/api/v1")
	v1.Use(middleware.Auth(s.config.APIKey))

	// Session handlers
	sessionHandler := handler.NewSessionHandler(s.sessionSvc)
	v1.POST("/session", sessionHandler.Create)
	v1.GET("/session", sessionHandler.List)
	v1.GET("/session/:id", sessionHandler.Get)
	v1.DELETE("/session/:id", sessionHandler.Delete)
	v1.POST("/session/:id/message", sessionHandler.Message)
	v1.POST("/session/:id/cancel", sessionHandler.Cancel)
	v1.GET("/session/:id/event", sessionHandler.SSE)

	// Artifact handlers
	artifactHandler := handler.NewArtifactHandler(s.sessionSvc)
	v1.GET("/session/:id/artifact", artifactHandler.List)
	v1.GET("/session/:id/artifact/:art_id", artifactHandler.Get)

	// Legacy routes (deprecated, for backward compat)
	v1.POST("/sessions", sessionHandler.Create)
	v1.GET("/sessions/:id", sessionHandler.Get)
	v1.POST("/sessions/:id/cancel", sessionHandler.Cancel)

	// Swagger UI (only in DevMode)
	if s.config.DevMode {
		s.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		s.log.Info("swagger ui enabled", "path", "/swagger/index.html")
	}

	// K8s health probe
	s.engine.GET("/healthz", handler.Health)
}
