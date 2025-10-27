package handlers

import (
	"net/http"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/The-Promised-Neverland/master-server/pkg/system"
	"github.com/gin-gonic/gin"
)

func (h *Handler) HealthCheck(c *gin.Context) {
	uptime := system.Uptime()
	health := models.HealthCheck{
		Status: "Healthy",
		Uptime: uptime,
	}
	resp := models.Message{
		Type:    "health_check",
		Payload: health,
	}
	c.JSON(http.StatusOK, resp)
}
