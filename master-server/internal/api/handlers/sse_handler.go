package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/The-Promised-Neverland/master-server/internal/sse"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SSEHandler struct {
	Hub     *sse.SSEHub
	Service interface {
		GetAllAgents() []*models.AgentInfo
		SendAgentListToFrontend()
	}
}

func NewSSEHandler(hub *sse.SSEHub) *SSEHandler {
	return &SSEHandler{
		Hub: hub,
	}
}

func (ssh *SSEHandler) SetService(service interface {
	GetAllAgents() []*models.AgentInfo
	SendAgentListToFrontend()
}) {
	ssh.Service = service
}

func (ssh *SSEHandler) StreamHandler(c *gin.Context) {
	connID := uuid.New().String()
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Credentials", "true")
	conn := ssh.Hub.Connect(connID)
	defer ssh.Hub.Disconnect(connID)
	connectedMsg := models.Message{
		Type:    "connected",
		Payload: map[string]string{"id": connID},
	}
	connectedData, _ := json.Marshal(connectedMsg)
	fmt.Fprintf(c.Writer, "data: %s\n\n", string(connectedData))
	c.Writer.Flush()
	if ssh.Service != nil {
		agents := ssh.Service.GetAllAgents()
		agentListMsg := models.Message{
			Type:    "agent_list",
			Payload: agents,
		}
		agentListData, _ := json.Marshal(agentListMsg)
		fmt.Fprintf(c.Writer, "data: %s\n\n", string(agentListData))
		c.Writer.Flush()
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case data, ok := <-conn.SendCh:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", string(data))
			c.Writer.Flush()
		case <-ticker.C:
			fmt.Fprintf(c.Writer, ": keepalive\n\n")
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

