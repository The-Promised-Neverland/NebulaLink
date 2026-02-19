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


// TODO: Processing of chunks is remaining
func (h *WSHub) DataStreamPump(c *Connection) {
	for {
		select {
		case chunk := <-c.StreamCh:
			fmt.Printf("Received chunk as string: %s\n", string(chunk))
			var statusMsg models.Message
			if err := json.Unmarshal(chunk, &statusMsg); err != nil {
				fmt.Printf("Failed to unmarshal message from %s: %v\n", c.Id, err)
				continue
			}
			if payload, ok := statusMsg.Payload.(map[string]interface{}); ok {
				if status, ok := payload["status"].(string); ok {
					if status == "completed" {
						c.StreamCh = make(chan []byte, 1024*64)
						fmt.Println("Stream has completed. Continuing to receive further data...")
					}
				}
			}
		case <-c.Ctx.Done():
			return
		}
	}
}

func (h *WSHub) ReadPump(c *Connection) {
	defer h.closeConnection(c)
	c.connMutex.RLock()
	if c.Conn == nil {
		c.connMutex.RUnlock()
		return
	}
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.connMutex.RUnlock()
	h.handlePong(c)
	for {
		select {
		case <-c.Ctx.Done():
			return
		default:
			c.connMutex.RLock()
			if c.Conn == nil {
				c.connMutex.RUnlock()
				return
			}
			conn := c.Conn
			c.connMutex.RUnlock()
			msgType, msgBytes, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("WebSocket read error for %s: %v\n", c.Id, err)
				}
				c.Cancel()
				return
			}
			switch msgType {
			case websocket.BinaryMessage:
				for {
					select {
					case c.StreamCh <- msgBytes:
					case <-c.Ctx.Done():
						return
					default:
						
					}
				}
			default:
				c.connMutex.RLock()
				if c.Conn != nil {
					c.Conn.SetReadDeadline(time.Now().Add(pongWait))
				}
				c.connMutex.RUnlock()
				h.Mutex.Lock()
				c.LastSeen = time.Now()
				h.Mutex.Unlock()
				var msg models.Message
				if err := json.Unmarshal(msgBytes, &msg); err != nil {
					fmt.Printf("Failed to unmarshal message from %s: %v\n", c.Id, err)
					continue
				}
				select {
				case c.IncomingCh <- msg:
				case <-c.Ctx.Done():
					return
				default:
					fmt.Printf("Incoming channel full for %s, dropping message\n", c.Id)
				}
			}
		}
	}
}

func (h *WSHub) WritePump(c *Connection) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		h.closeConnection(c)
	}()
	for {
		select {
		case msg, ok := <-c.SendCh:
			if !ok {
				c.connMutex.RLock()
				if c.Conn != nil {
					c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				}
				c.connMutex.RUnlock()
				return
			}
			c.connMutex.RLock()
			if c.Conn == nil {
				c.connMutex.RUnlock()
				return
			}
			conn := c.Conn
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.connMutex.RUnlock()
			data, err := json.Marshal(msg)
			if err != nil {
				fmt.Printf("Marshal error for %s: %v\n", c.Id, err)
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				fmt.Printf("Send failed to %s: %v\n", c.Id, err)
				return
			}
		case <-ticker.C:
			c.connMutex.RLock()
			if c.Conn == nil {
				c.connMutex.RUnlock()
				return
			}
			conn := c.Conn
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.connMutex.RUnlock()
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Printf("Ping failed to %s: %v\n", c.Id, err)
				return
			}
		case <-c.Ctx.Done():
			return
		}
	}
}

// Broadcast message from agent to all frontend clients
func (h *WSHub) BroadcasterPump(c *Connection) {
	for {
		select {
		case msg, ok := <-c.IncomingCh:
			if !ok {
				return
			}
			select {
			case <-c.Ctx.Done():
				return
			default:
				h.SSEHub.Broadcast(msg)
			}
		case <-c.Ctx.Done():
			return
		}
	}
}
