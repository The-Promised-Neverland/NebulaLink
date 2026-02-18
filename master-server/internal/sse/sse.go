package sse

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
)

type Connection struct {
	ID       string
	SendCh   chan []byte
	LastSeen time.Time
}

type SSEHub struct {
	Connections map[string]*Connection
	Mutex       sync.RWMutex
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		Connections: make(map[string]*Connection),
	}
}

func (h *SSEHub) Connect(id string) *Connection {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()
	conn := &Connection{
		ID:       id,
		SendCh:   make(chan []byte, 100),
		LastSeen: time.Now(),
	}
	h.Connections[id] = conn
	fmt.Printf("SSE connection established: %s\n", id)
	return conn
}

func (h *SSEHub) Disconnect(id string) {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()
	if conn, exists := h.Connections[id]; exists {
		close(conn.SendCh)
		delete(h.Connections, id)
		fmt.Printf("SSE connection closed: %s\n", id)
	}
}

func (h *SSEHub) Broadcast(msg models.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("Failed to marshal SSE message: %v\n", err)
		return
	}
	h.Mutex.RLock()
	connections := make([]*Connection, 0, len(h.Connections))
	for _, conn := range h.Connections {
		connections = append(connections, conn)
	}
	h.Mutex.RUnlock()
	for _, conn := range connections {
		select {
		case conn.SendCh <- data:
		default:
			fmt.Printf("SSE send channel full for %s, dropping message\n", conn.ID)
		}
	}
}


