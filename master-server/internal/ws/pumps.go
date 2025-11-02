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
	defer h.closeConnection(c)
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	h.handlePong(c)
	for {
		select {
		case <-c.Ctx.Done():
			return
		default:
			_, msgBytes, err := c.Conn.ReadMessage()
			if err != nil {
                c.Cancel()
                return
            }
			h.Mutex.Lock()
			c.LastSeen = time.Now()
			h.Mutex.Unlock()
			var msg models.Message
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				fmt.Printf("❌ Failed to unmarshal message from %s: %v\n", c.Role, err)
				continue
			}
			c.IncomingCh <- msg
		}
	}
}

func (h *Hub) WritePump(c *Connection) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		h.closeConnection(c)
	}()
	for {
		select {
		case msg, ok := <-c.SendCh:
			if !ok {
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			data, err := json.Marshal(msg)
			if err != nil {
				fmt.Printf("❌ Marshal error for %s: %v\n", c.Role, err)
				continue
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				fmt.Printf("⚠️ Send failed to %s: %v\n", c.Role, err)
				return
			}
		case <-c.Ctx.Done():
			return
		}
	}
}

func (h *Hub) ProcessorPump(c *Connection) {
	for msg := range c.IncomingCh {
		h.MssgProcessor.ProcessMessage(c.Role, &msg)
	}
}
