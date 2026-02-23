package ws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/The-Promised-Neverland/master-server/internal/sse"
	"github.com/The-Promised-Neverland/master-server/internal/transfer"
	"github.com/gorilla/websocket"
)

type WSHub struct {
	Connections     map[string]*Connection
	Mutex           sync.RWMutex
	SSEHub          *sse.SSEHub
	TransferManager *transfer.TransferManager
	Handlers        map[string]func(msg *models.Message, connection *Connection) error
}

func NewWSHub(sseHub *sse.SSEHub) *WSHub {
	hub := &WSHub{
		Connections: make(map[string]*Connection),
		SSEHub:      sseHub,
		Handlers:    make(map[string]func(msg *models.Message, connection *Connection) error),
	}
	hub.TransferManager = transfer.NewTransferManager(hub, hub)
	return hub
}

// Send implements transfer.MessageSender interface
func (h *WSHub) Send(agentID string, msg transfer.Outbound) {
	h.Mutex.RLock()
	c := h.Connections[agentID]
	h.Mutex.RUnlock()
	if c == nil || c.Conn == nil {
		return
	}
	select {
	case c.SendCh <- msg:
	default:
		fmt.Printf("Send channel full for %s\n", agentID)
	}
}

func (h *WSHub) GetConnection(agentID string) transfer.ConnectionInfo {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()
	return h.Connections[agentID]
}

func (h *WSHub) RegisterHandler(msgType string, handler func(msg *models.Message, connection *Connection) error) {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()
	h.Handlers[msgType] = handler
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
		h.ProcessorPump(connection)
	}()
	go func() {
		defer connection.wg.Done()
		h.DataStreamPump(connection) // Start stream processing
	}()
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
