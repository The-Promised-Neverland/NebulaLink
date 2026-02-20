package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 2 * 1024 * 1024 // 2MB
)

// TODO: Pass on the chunks to the requesting agent
func (h *WSHub) DataStreamPump(c *Connection) {
	for {
		select {
		case chunk := <-c.StreamCh:
			fmt.Printf("Transfer in progress... %s -> %s: %d bytes\n", c.Id, c.RelayTo, len(chunk))
			// TODO: Need to pass this onto the relayTo agent
			sendToAgent := h.Connections[c.RelayTo].Id
			h.Send(sendToAgent, Outbound{Binary: chunk})
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
				select {
				case c.StreamCh <- msgBytes:
				case <-c.Ctx.Done():
				case <-time.After(time.Second * 5): // Handling backpressure in lazy way
					fmt.Println("Timed out trying to send message to StreamCh")
					return
				}
			case websocket.TextMessage:
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
				case c.IncomingCh <- Outbound{Msg: &msg}:
				case <-c.Ctx.Done():
					return
				default:
					// TODO: Handling backpressure
					fmt.Printf("Incoming channel full for %s, dropping message\n", c.Id)
				}
			default:
			}
		}
	}
}

// TODO: NEED TO MOPDIFY THIS TO SEND AND ACCEPT BYTES/TEXT
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
			if msg.Msg != nil {
				bytes, err := json.Marshal(*msg.Msg)
				if err != nil {
					fmt.Printf("Marshal error for %s: %v\n", c.Id, err)
					continue
				}
				fmt.Printf("Sending to %s: %s\n", c.Id, string(bytes))
				if err := conn.WriteMessage(websocket.TextMessage, bytes); err != nil {
					fmt.Printf("TEXT: Send failed to %s: %v\n", c.Id, err)
					return
				}
			} else {
				if err := conn.WriteMessage(websocket.BinaryMessage, msg.Binary); err != nil {
					fmt.Printf("BINARY: Send failed to %s: %v\n", c.Id, err)
					return
				}
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
				msgRecieved := *msg.Msg
				if msgRecieved.Type == "master_filesystem_request" {
					payloadMap, ok := msgRecieved.Payload.(map[string]interface{})
					if ok {
						if relayTo, ok2 := payloadMap["requesting_agent_id"].(string); ok2 {
							c.RelayTo = relayTo
						} else {
							fmt.Println("requesting_agent_id not found or not a string")
						}
					} else {
						fmt.Println("Payload is not map[string]interface{}")
					}
				}
				h.SSEHub.Broadcast(msgRecieved) // Broadcasts to all frontend clients via SSE
			}
		case <-c.Ctx.Done():
			return
		}
	}
}
