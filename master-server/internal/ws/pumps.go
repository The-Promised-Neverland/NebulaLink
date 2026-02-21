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
	pongWait       = 60 * time.Second
	pingPeriod     = 30 * time.Second
	writeWait      = 10 * time.Second
)

// TODO: Pass on the chunks to the requesting agent
func (h *WSHub) DataStreamPump(c *Connection) {
	for {
		select {
		case chunk := <-c.StreamCh:
			fmt.Printf("Transfer in progress... %s -> %s: %d bytes\n", c.Id, c.RelayTo, len(chunk))
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
				if msgRecieved.Type == models.MasterMsgRelayManager {
					payloadMap, ok := msgRecieved.Payload.(map[string]interface{})
					if ok {
						if status, hasStatus := payloadMap["status"].(string); hasStatus && status != "" {
							if c.RelayTo != "" {
								// Don't forward "initiated" if we already sent it from master
								if status == "initiated" && c.InitiatedSent {
									fmt.Printf("Skipping duplicate 'initiated' status from source agent %s (already sent by master)\n", c.Id)
									continue
								}
								statusMsg := models.Message{
									Type: models.MasterMsgRelayManager,
									Payload: map[string]interface{}{
										"status":   status,
										"agent_id": c.Id, // Source agent (the one sending files)
									},
								}
								h.Send(c.RelayTo, Outbound{Msg: &statusMsg})
								if status == "initiated" {
									c.InitiatedSent = true
								}
								fmt.Printf("Forwarded '%s' status to destination agent %s from source agent %s\n", status, c.RelayTo, c.Id)
							}
						}
					}
				}
				if msgRecieved.Type == models.MasterMsgAgentRequestFile {
					payloadMap, ok := msgRecieved.Payload.(map[string]interface{})
					if ok {
						if requestingAgentID, ok2 := payloadMap["requesting_agent_id"].(string); ok2 && requestingAgentID != "" {
							if c.RelayTo == "" {
								c.RelayTo = requestingAgentID
								c.InitiatedSent = false // Reset for new transfer
								fmt.Printf("Set RelayTo=%s for source agent %s\n", requestingAgentID, c.Id)
							}
						}
					}
				}
				if msgRecieved.Type == "agent_metrics" {
					if payloadMap, ok := msgRecieved.Payload.(map[string]interface{}); ok {
						if endpoint, hasEndpoint := payloadMap["public_endpoint"].(string); hasEndpoint && endpoint != "" {
							h.Mutex.Lock()
							c.PublicEndpoint = endpoint
							h.Mutex.Unlock()
							delete(payloadMap, "public_endpoint")
							delete(payloadMap, "nat_type")
							msgRecieved.Payload = payloadMap
						}
					}
				}
				h.SSEHub.Broadcast(msgRecieved) // Broadcasts to all frontend clients via SSE
			}
		case <-c.Ctx.Done():
			return
		}
	}
}
