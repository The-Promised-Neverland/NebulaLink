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

func (h *Hub) ReadPump(c *Connection) {
	defer func() {
		if err := c.Conn.Close(); err != nil {
			fmt.Printf("Agent %s disconnected\n", c.Role)
		}
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	h.handlePong(c)
	for {
		select {
		case <-c.DisconnectCh:
			return
		default:
			_, msgBytes, err := c.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("WebSocket error for %s: %v\n", c.Role, err)
				}
				return
			}
			c.Conn.SetReadDeadline(time.Now().Add(pongWait))
			h.Mutex.Lock()
			c.LastSeen = time.Now()
			h.Mutex.Unlock()
			var msg models.Message
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				fmt.Printf("Failed to unmarshal message from %s: %v\n", c.Role, err)
				continue
			}
			c.IncomingCh <- msg
		}
	}
}

func (h *Hub) WritePump(c *Connection) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer func() {
		c.Conn.Close()
		fmt.Printf("Agent %s write pump stopped\n", c.Role)
	}()
	for {
		select {
		case msg, ok := <-c.SendCh:
			if !ok {
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			data, err := json.Marshal(msg)
			if err != nil {
				fmt.Printf("Failed to marshal message for %s: %v\n", c.Role, err)
				continue
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				fmt.Printf("Failed to send message to %s: %v\n", c.Role, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := h.sendPingToAgent(c); err != nil {
				fmt.Printf("⚠️ Ping failed for %s: %v\n", c.Role, err)
				return
			}

		case <-c.DisconnectCh:
			return
		}
	}
}

func (h *Hub) ProcessorPump(c *Connection) {
	for msg := range c.IncomingCh {
		h.MssgProcessor.ProcessMessage(c.Role, &msg)
	}
}
