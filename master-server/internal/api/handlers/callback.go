package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/gin-gonic/gin"
)

func (h *Handler) ReceiveAgentMetrics(c *gin.Context) {
	agentID := c.Param("id")
	var msg models.Message
	if err := c.BindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var metrics models.Metrics
	payloadBytes, _ := json.Marshal(msg.Payload)
	if err := json.Unmarshal(payloadBytes, &metrics); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metrics payload"})
		return
	}
	metrics.AgentID = agentID
	h.Mutex.Lock()
	ch, exists := h.PendingRequests[agentID]
	if exists {
		ch <- metrics
		close(ch)
		delete(h.PendingRequests, agentID)
	}
	h.Mutex.Unlock()
	c.JSON(http.StatusOK, gin.H{"success": true})
}
