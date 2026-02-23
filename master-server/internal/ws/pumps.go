package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/The-Promised-Neverland/master-server/internal/transfer"
	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 2 * 1024 * 1024 // 2MB
	pongWait       = 60 * time.Second
	pingPeriod     = 30 * time.Second
	writeWait      = 10 * time.Second
)

func (h *WSHub) DataStreamPump(c *Connection) {
	for {
		select {
		case chunk := <-c.StreamCh:
			if c.RelayTo == "" {
				continue
			}
			fmt.Printf("Transfer in progress... %s -> %s: %d bytes\n", c.Id, c.RelayTo, len(chunk))
			h.Mutex.RLock()
			destConn := h.Connections[c.RelayTo]
			h.Mutex.RUnlock()
			if destConn == nil {
				continue
			}
			select {
			case destConn.SendCh <- transfer.Outbound{Binary: chunk}:
			default:
				fmt.Printf("Send channel full for %s\n", c.RelayTo)
			}
		case <-c.Ctx.Done():
			return
		}
	}
}

func (h *WSHub) ReadPump(c *Connection) {
	defer h.closeConnection(c)
	c.ConnMutex.RLock()
	if c.Conn == nil {
		c.ConnMutex.RUnlock()
		return
	}
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.ConnMutex.RUnlock()
	for {
		select {
		case <-c.Ctx.Done():
			return
		default:
			c.ConnMutex.RLock()
			if c.Conn == nil {
				c.ConnMutex.RUnlock()
				return
			}
			conn := c.Conn
			c.ConnMutex.RUnlock()
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
				c.ConnMutex.RLock()
				if c.Conn != nil {
					c.Conn.SetReadDeadline(time.Now().Add(pongWait))
				}
				c.ConnMutex.RUnlock()
				h.Mutex.Lock()
				c.LastSeen = time.Now()
				h.Mutex.Unlock()
				var msg models.Message
				if err := json.Unmarshal(msgBytes, &msg); err != nil {
					fmt.Printf("Failed to unmarshal message from %s: %v\n", c.Id, err)
					continue
				}
				select {
				case c.IncomingCh <- transfer.Outbound{Msg: &msg}:
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
				c.ConnMutex.RLock()
				if c.Conn != nil {
					c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				}
				c.ConnMutex.RUnlock()
				return
			}
			c.ConnMutex.RLock()
			if c.Conn == nil {
				c.ConnMutex.RUnlock()
				return
			}
			conn := c.Conn
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.ConnMutex.RUnlock()
			if msg.Msg != nil {
				bytes, err := json.Marshal(*msg.Msg)
				if err != nil {
					fmt.Printf("Marshal error for %s: %v\n", c.Id, err)
					continue
				}
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
			c.ConnMutex.RLock()
			if c.Conn == nil {
				c.ConnMutex.RUnlock()
				return
			}
			conn := c.Conn
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.ConnMutex.RUnlock()
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Printf("Ping failed to %s: %v\n", c.Id, err)
				return
			}
		case <-c.Ctx.Done():
			return
		}
	}
}

func (h *WSHub) ProcessorPump(c *Connection) {
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
				if msg.Msg == nil {
					continue
				}
				msgReceived := *msg.Msg
				h.Mutex.RLock()
				handler, hasHandler := h.Handlers[msgReceived.Type]
				h.Mutex.RUnlock()
				if hasHandler {
					if err := handler(&msgReceived, c); err != nil {
						fmt.Printf("Handler error for message type %s from %s: %v\n", msgReceived.Type, c.Id, err)
					}
				} else {
					fmt.Printf("No handler registered for message type: %s from %s\n", msgReceived.Type, c.Id)
				}
				if h.SSEHub != nil {
					h.SSEHub.Broadcast(msgReceived)
				}
			}
		case <-c.Ctx.Done():
			return
		}
	}
}
