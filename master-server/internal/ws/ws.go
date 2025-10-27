package ws

import (
	"fmt"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/gorilla/websocket"
)

type AgentConnection struct {
	Conn         *websocket.Conn
	OS           string
	LastSeen     time.Time
	DisconnectCh chan struct{}
	SendCh       chan models.Message
	IncomingCh   chan models.Message
}

type Hub struct {
	Agents  map[string]*AgentConnection
	Mutex   sync.RWMutex
	Handler func(msgType string, payload any)
}

// NewHub initializes the hub
func NewHub(handler func(msgType string, payload any)) *Hub {
	return &Hub{
		Agents:  make(map[string]*AgentConnection),
		Handler: handler,
	}
}

// RegisterConnection registers or reconnects an agent
func (h *Hub) RegisterConnection(id string, conn *websocket.Conn) {
	h.Mutex.Lock()
	agent, exists := h.Agents[id]
	if exists {
		fmt.Printf("♻️  Agent %s reconnecting\n", id)
		close(agent.DisconnectCh)
		if agent.Conn != nil {
			agent.Conn.Close()
		}
		time.Sleep(100 * time.Millisecond)
		agent.Conn = conn
		agent.LastSeen = time.Now()
		agent.DisconnectCh = make(chan struct{})
	} else {
		fmt.Printf("✨ Agent %s registered (new)\n", id)
		agent = &AgentConnection{
			Conn:         conn,
			LastSeen:     time.Now(),
			DisconnectCh: make(chan struct{}),
			SendCh:       make(chan models.Message, 100),
			IncomingCh:   make(chan models.Message, 500),
		}
		h.Agents[id] = agent
	}
	h.Mutex.Unlock()
	go h.ReadPump(id, agent)
	go h.WritePump(id, agent)
	go h.dispatchPump(agent)
}

// SendToAgent sends a message to a specific agent
func (h *Hub) SendToAgent(agentID string, msg models.Message) error {
	h.Mutex.RLock()
	agent, exists := h.Agents[agentID]
	h.Mutex.RUnlock()
	if !exists || agent.Conn == nil {
		return fmt.Errorf("agent %s not connected", agentID)
	}
	select {
	case agent.SendCh <- msg:
		return nil
	default:
		return fmt.Errorf("agent %s send channel full", agentID)
	}
}
