package ws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/The-Promised-Neverland/master-server/internal/sse"
	"github.com/gorilla/websocket"
)

type WSHub struct {
	Connections map[string]*Connection
	Mutex       sync.RWMutex
	SSEHub      *sse.SSEHub
}

func NewWSHub(sseHub *sse.SSEHub) *WSHub {
	hub := &WSHub{
		Connections: make(map[string]*Connection),
		SSEHub:      sseHub,
	}
	return hub
}

// Registers or re-connects an agent
func (h *WSHub) Connect(name string, id string, os string, conn *websocket.Conn) {
	h.Mutex.Lock()
	var connection *Connection
	if existing, exists := h.Connections[id]; exists {
		fmt.Printf("Reconnecting: %s\n", id)
		existing.Cancel()
		if existing.Conn != nil {
			existing.Conn.Close()
		}
		existing.wg.Wait()
		existing.Conn = conn
		existing.LastSeen = time.Now()
		existing.Name = name
		if os != "" {
			existing.OS = os
		}
		ctx, cancel := context.WithCancel(context.Background())
		existing.Ctx = ctx
		existing.Cancel = cancel
		connection = existing
	} else {
		fmt.Printf("New connection: %s\n", id)
		connection = NewConnection(name, id, os, conn)
		h.Connections[id] = connection
	}
	h.Mutex.Unlock()
	connection.wg.Add(4)
	go func() {
		defer connection.wg.Done()
		h.ReadPump(connection)
	}()
	go func() {
		defer connection.wg.Done()
		h.WritePump(connection)
	}()
	go func() {
		defer connection.wg.Done()
		h.BroadcasterPump(connection)
	}()
	go func() {
		defer connection.wg.Done()
		h.DataStreamPump(connection) // Start stream processing
	}()
}

func (h *WSHub) Send(agentID string, msg Outbound) {
	h.Mutex.RLock()
	c := h.Connections[agentID]
	h.Mutex.RUnlock()
	if c == nil || c.Conn == nil {
		return
	}
	if msg.Msg != nil && msg.Msg.Type == "master_filesystem_request" {
		if payloadMap, ok := msg.Msg.Payload.(map[string]interface{}); ok {
			if requestedAgentID, ok2 := payloadMap["requestedAgentID"].(string); ok2 && requestedAgentID != "" {
				if c.RelayTo == "" {
					c.RelayTo = requestedAgentID
					initiatedMsg := models.Message{
						Type: "master_filesystem_request",
						Payload: map[string]interface{}{
							"status":   "initiated",
							"agent_id": agentID,
						},
					}
					h.Send(requestedAgentID, Outbound{Msg: &initiatedMsg})
					fmt.Printf("Set RelayTo=%s for agent %s and sent 'initiated' message to %s\n", requestedAgentID, agentID, requestedAgentID)
				}
			}
		}
	}
	select {
	case c.SendCh <- msg:
	default:
		fmt.Printf("Send channel full for %s\n", agentID)
	}
}

func (h *WSHub) closeConnection(c *Connection) {
	c.Cancel()
	if c.Conn != nil {
		_ = c.Conn.Close()
	}
	h.Mutex.Lock()
	c.LastSeen = time.Now()
	c.Conn = nil
	h.Mutex.Unlock()
	msg := models.Message{
		Type: "agent_disconnected",
		Payload: map[string]string{
			"agent_id": c.Id,
		},
	}
	if h.SSEHub != nil {
		h.SSEHub.Broadcast(msg)
	}
	fmt.Printf("Disconnected: %s (Last seen %v)\n", c.Id, c.LastSeen)
}
