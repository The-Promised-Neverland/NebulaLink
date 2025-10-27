package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 8192
)

func (h *Hub) ReadPump(id string, agent *AgentConnection) {
	defer func() {
		if err := agent.Conn.Close(); err != nil {
			fmt.Printf("Agent %s disconnected\n", id)
		}
	}()
	agent.Conn.SetReadLimit(maxMessageSize)
	agent.Conn.SetReadDeadline(time.Now().Add(pongWait))
	h.handlePong(agent)
	for {
		select {
		case <-agent.DisconnectCh:
			return
		default:
			_, msgBytes, err := agent.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("WebSocket error for %s: %v\n", id, err)
				}
				return
			}
			agent.Conn.SetReadDeadline(time.Now().Add(pongWait))
			h.Mutex.Lock()
			agent.LastSeen = time.Now()
			h.Mutex.Unlock()
			var msg models.Message
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				fmt.Printf("Failed to unmarshal message from %s: %v\n", id, err)
				continue
			}
			agent.IncomingCh <- msg
		}
	}
}

func (h *Hub) WritePump(id string, agent *AgentConnection) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer func() {
		agent.Conn.Close()
		fmt.Printf("Agent %s write pump stopped\n", id)
	}()
	for {
		select {
		case msg, ok := <-agent.SendCh:
			if !ok {
				return
			}
			agent.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			data, err := json.Marshal(msg)
			if err != nil {
				fmt.Printf("Failed to marshal message for %s: %v\n", id, err)
				continue
			}
			if err := agent.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				fmt.Printf("Failed to send message to %s: %v\n", id, err)
				return
			}

		case <-ticker.C:
			agent.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := h.sendPingToAgent(agent); err != nil {
				fmt.Printf("⚠️ Ping failed for %s: %v\n", id, err)
				return
			}

		case <-agent.DisconnectCh:
			return
		}
	}
}

func (h *Hub) dispatchPump(agent *AgentConnection) {
	for msg := range agent.IncomingCh {
		h.Handler(msg.Type, msg.Payload)
	}
}
