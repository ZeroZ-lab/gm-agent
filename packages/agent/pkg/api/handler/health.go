package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gm-agent-org/gm-agent/pkg/api/dto"
)

// Health godoc
// @Summary      Health check
// @Description  Returns server health and version
// @Tags         global
// @Produce      json
// @Success      200 {object} dto.HealthResponse
// @Router       /health [get]
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, dto.HealthResponse{
		Status:  "healthy",
		Version: "1.0.0",
	})
}
