package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/The-Promised-Neverland/master-server/internal/service"
	"github.com/The-Promised-Neverland/master-server/pkg/system"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	Service         *service.Service
	PendingRequests map[string]chan interface{}
	Mutex           sync.RWMutex
}

func NewHandler(s *service.Service) *Handler {
	return &Handler{
		Service:         s,
		PendingRequests: make(map[string]chan interface{}),
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
		select {
		case ch <- metrics:
		default:
			fmt.Printf("Recieve Timeout")
		}
		delete(h.PendingRequests, agentID)
		close(ch)
	}
	h.Mutex.Unlock()
	c.JSON(http.StatusOK, gin.H{"success": true})
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

func (h *Handler) GetAgentMetrics(c *gin.Context) {
	agentID := c.Param("id")
	responseCh := make(chan interface{}, 1)
	h.Mutex.Lock()
	h.PendingRequests[agentID] = responseCh
	h.Mutex.Unlock()
	h.Service.TriggerAgentforMetrics(agentID)
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
