package handlers

import (
	"net/http"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/gin-gonic/gin"
)

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

func (h *Handler) GetAgentMetrics(c *gin.Context) {
	agentID := c.Param("id")
	responseCh := make(chan interface{}, 1)
	h.Mutex.Lock()
	h.PendingRequests[agentID] = responseCh
	h.Mutex.Unlock()
	if err := h.Service.TriggerAgentforMetrics(agentID); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "failed to reach agent"})
		return
	}
	select {
	case agentMetrics := <-responseCh:
		resp := models.Message{
			Type:    "metrics_request",
			Payload: agentMetrics,
		}
		c.JSON(http.StatusOK, resp)
	case <-time.After(30 * time.Second):
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "agent timeout"})
		h.Mutex.Lock()
		delete(h.PendingRequests, agentID)
		h.Mutex.Unlock()
	}
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
