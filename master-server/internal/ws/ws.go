package ws

import (
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
func (h *Hub) Connect(role string, conn *websocket.Conn) {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()
	if existing, exists := h.Connections[role]; exists {
		fmt.Printf("Reconnecting: %s (closing old connection)\n", role)
		if existing.Conn != nil {
			existing.Cancel()
			existing.Conn.Close()
		}
		delete(h.Connections, role)
	} else {
		fmt.Printf("New connection: %s\n", role)
	}
	connection := NewConnection(role, conn)
	h.Connections[role] = connection
	go h.ReadPump(connection)
	go h.WritePump(connection)
	go h.ProcessorPump(connection)
}

func (h *Hub) routeMessages() {
	for routedMsg := range h.MssgProcessor.OutgoingCh {
		h.Send(routedMsg.Target, routedMsg.Message)
	}
}

func (h *Hub) Send(id string, msg models.Message) {
	h.Mutex.RLock()
	c := h.Connections[id]
	h.Mutex.RUnlock()
	if c == nil || c.Conn == nil {
		return
	}
	select {
	case c.SendCh <- msg:
	default:
		fmt.Printf("Send channel full for %s\n", id)
	}
}

func (h *Hub) closeConnection(c *Connection) {
	c.Cancel()
	if c.Conn != nil {
		_ = c.Conn.Close()
	}
	h.Mutex.Lock()
	c.LastSeen = time.Now()
	delete(h.Connections, c.Role)
	h.Mutex.Unlock()

	fmt.Printf("Disconnected: %s (Last seen %v)\n", c.Role, c.LastSeen)
}
