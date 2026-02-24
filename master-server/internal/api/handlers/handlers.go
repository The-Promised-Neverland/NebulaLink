package handlers

import (
	"net/http"
	"sync"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/The-Promised-Neverland/master-server/internal/service"
	"github.com/The-Promised-Neverland/master-server/pkg/system"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service *service.Service
	Mutex   sync.RWMutex
}

func NewHandler(s *service.Service) *Handler {
	return &Handler{
		Service: s,
	}
}

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

func (h *Handler) ListAgents(c *gin.Context) {
	agentsInfo := h.Service.GetAllAgents()
	resp := models.Message{
		Type:    "agent_list",
		Payload: agentsInfo,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetAgent(c *gin.Context) {
	agentID := c.Param("id")
	agentInfo := h.Service.GetAgent(agentID)
	resp := models.Message{
		Type:    "agent_info",
		Payload: agentInfo,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) TriggerAgentMetrics(c *gin.Context) {
	agentID := c.Param("id")
	h.Service.TriggerAgentforMetrics(agentID)
	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"message": "Metrics triggered for agent " + agentID,
	})
}

func (h *Handler) RestartAgent(c *gin.Context) {
	agentID := c.Param("id")
	h.Service.RestartAgent(agentID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Agent restart initiated",
	})
}

func (h *Handler) UninstallAgent(c *gin.Context) {
	agentID := c.Param("id")
	h.Service.UninstallAgent(agentID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Agent uninstallation initiated",
	})
}

func (h *Handler) GetAgentFileSystem(c *gin.Context) {
	requestingAgentID := c.Param("id")           // Agent that wants to receive the file (requesting agent)
	sourceAgentID := c.Param("getFromAgent")     // Agent that has the file (source agent)
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Path binding error: " + err.Error(),
		})
		return
	}
	agentOnlineStatus, err := h.Service.IsAgentOnline(sourceAgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if !agentOnlineStatus {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Requested agent is offline. Please ensure the agent is online to fetch the filesystem data.",
		})
		return
	}
	h.Service.GetAgentFileSystem(requestingAgentID, sourceAgentID, req.Path)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Requested file will be available in your shared folder shortly",
	})
}
