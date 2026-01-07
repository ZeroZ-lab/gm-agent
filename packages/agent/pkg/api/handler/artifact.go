package handler

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gm-agent-org/gm-agent/pkg/api/dto"
	"github.com/gm-agent-org/gm-agent/pkg/api/service"
)

type ArtifactHandler struct {
	svc *service.SessionService
}

func NewArtifactHandler(svc *service.SessionService) *ArtifactHandler {
	return &ArtifactHandler{svc: svc}
}

// List godoc
// @Summary      List session artifacts
// @Description  List all artifacts produced by the session
// @Tags         artifact
// @Produce      json
// @Param        id path string true "Session ID"
// @Success      200 {object} map[string]any "{"artifacts": []Artifact}"
// @Failure      404 {object} dto.ErrorResponse
// @Router       /api/v1/session/{id}/artifact [get]
func (h *ArtifactHandler) List(c *gin.Context) {
	sessionID := c.Param("id")
	artifacts, err := h.svc.ListArtifacts(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"artifacts": artifacts})
}

// Get godoc
// @Summary      Get artifact content
// @Description  Download artifact content or get metadata
// @Tags         artifact
// @Produce      json, octet-stream
// @Param        id path string true "Session ID"
// @Param        art_id path string true "Artifact ID"
// @Success      200 {object} interface{} "File content or Artifact metadata"
// @Failure      404 {object} dto.ErrorResponse
// @Router       /api/v1/session/{id}/artifact/{art_id} [get]
func (h *ArtifactHandler) Get(c *gin.Context) {
	sessionID := c.Param("id")
	artifactID := c.Param("art_id")

	art, err := h.svc.GetArtifact(sessionID, artifactID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Error: err.Error()})
		return
	}

	// If file, try serve file
	if art.Type == "file" && art.Path != "" {
		if _, err := os.Stat(art.Path); err == nil {
			http.ServeFile(c.Writer, c.Request, art.Path)
			return
		}
	}

	c.JSON(http.StatusOK, art)
}
