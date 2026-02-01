package ws

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const (
	pongWait       = 60 * time.Second
	pingPeriod     = 30 * time.Second
	writeWait      = 10 * time.Second
)

func (h *Hub) sendPingToAgent(agent *Connection) error {
	if agent.Conn == nil {
		return fmt.Errorf("connection is nil")
	}
	if err := agent.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		fmt.Printf("Ping failed for %v\n", err)
		return err
	}
	return nil
}

func (h *Hub) handlePong(agent *Connection) {
	if agent.Conn == nil {
		return
	}
	agent.Conn.SetPongHandler(func(string) error {
		agent.Conn.SetReadDeadline(time.Now().Add(pongWait))
		h.Mutex.Lock()
		agent.LastSeen = time.Now()
		h.Mutex.Unlock()
		return nil
	})
}
