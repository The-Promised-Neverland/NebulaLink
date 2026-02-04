package ws

import (
	"context"
	"fmt"
	"sync"
	"time"

	ws_mssg_processor "github.com/The-Promised-Neverland/master-server/internal/api/processors"
	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/gorilla/websocket"
)

type Hub struct {
	Connections   map[string]*Connection
	Mutex         sync.RWMutex
	MssgProcessor *ws_mssg_processor.Processor
}

// NewHub initializes the hub
func NewHub() *Hub {
	processor := ws_mssg_processor.NewProcessor()
	hub := &Hub{
		Connections:   make(map[string]*Connection),
		MssgProcessor: processor,
	}
	go hub.routeMessages()
	return hub
}

// Registers or re-connects an agent
func (h *Hub) Connect(name string, id string, conn *websocket.Conn) {
	h.Mutex.Lock()
	var connection *Connection
	if existing, exists := h.Connections[id]; exists {
		fmt.Printf("Reconnecting: %s\n", id)
		if existing.Conn != nil {
			existing.Cancel()
			existing.Conn.Close()
		}
		existing.Conn = conn
		existing.LastSeen = time.Now()
		existing.Name = name
		ctx, cancel := context.WithCancel(context.Background())
		existing.Ctx = ctx
		existing.Cancel = cancel
		connection = existing
	} else {
		fmt.Printf("New connection: %s\n", id)
		connection = NewConnection(name, id, conn)
		h.Connections[id] = connection
	}
	h.Mutex.Unlock()
	go h.ReadPump(connection)
	go h.WritePump(connection)
	go h.ProcessorPump(connection)
	if name == "frontend" {
		h.sendAgentListToFrontend()
	}
}

func (h *Hub) routeMessages() {
	for routedMsg := range h.MssgProcessor.OutgoingCh {
		h.Send(routedMsg.TargetId, routedMsg.Message)
	}
}

func (h *Hub) Send(TargetId string, msg models.Message) {
	h.Mutex.RLock()
	c := h.Connections[TargetId]
	h.Mutex.RUnlock()
	if c == nil || c.Conn == nil {
		return
	}
	select {
	case c.SendCh <- msg:
	default:
		fmt.Printf("Send channel full for %s\n", TargetId)
	}
}

func (h *Hub) closeConnection(c *Connection) {
	c.Cancel()
	if c.Conn != nil {
		_ = c.Conn.Close()
	}
	h.Mutex.Lock()
	c.LastSeen = time.Now()
	c.Conn = nil
	isAgent := c.Id != "frontend" && c.Id != ""
	h.Mutex.Unlock()
	if isAgent {
		msg := models.Message{
			Type: "agent_disconnected",
			Payload: map[string]string{
				"agent_id": c.Id,
			},
		}
		h.Send("frontend", msg)
	}
	fmt.Printf("Disconnected: %s (Last seen %v)\n", c.Id, c.LastSeen)
}

// sendAgentListToFrontend sends the current agent list to the frontend
func (h *Hub) sendAgentListToFrontend() {
	h.Mutex.RLock()
	agents := make([]*models.AgentInfo, 0, len(h.Connections))
	for id, agent := range h.Connections {
		if id == "frontend" {
			continue
		}
		info := &models.AgentInfo{
			AgentID:  id,
			Name:     agent.Name,
			OS:       agent.OS,
			LastSeen: agent.LastSeen,
		}
		agents = append(agents, info)
	}
	h.Mutex.RUnlock()
	msg := models.Message{
		Type:    "agent_list",
		Payload: agents,
	}
	h.Send("frontend", msg)
}
