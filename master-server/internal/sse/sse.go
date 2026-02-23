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
	BroadCastCh chan models.Message
	Mutex       sync.RWMutex
}

func NewSSEHub() *SSEHub {
	h := &SSEHub{
		Connections: make(map[string]*Connection),
		BroadCastCh: make(chan models.Message, 100),
	}

	go h.Broadcastpump()
	return h
}

func (h *SSEHub) Broadcastpump() {
	for msg := range h.BroadCastCh {
		data, err := json.Marshal(msg)
		if err != nil {
			fmt.Printf("Failed to marshal SSE message: %v\n", err)
			continue
		}
		h.Mutex.RLock()
		for _, conn := range h.Connections {
			select {
			case conn.SendCh <- data:
			default:
				fmt.Printf("SSE send channel full for %s, dropping message\n", conn.ID)
			}
		}
		h.Mutex.RUnlock()
	}
}

func (h *SSEHub) Broadcast(msg models.Message) {
	select {
	case h.BroadCastCh <- msg:
	default:
		fmt.Printf("SSE broadcast channel full, dropping message\n")
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