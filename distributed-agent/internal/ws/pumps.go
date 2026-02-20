package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/gorilla/websocket"
)

const (
	readDeadline  = 70 * time.Second
	writeDeadline = 10 * time.Second
	pongWait      = 60 * time.Second
)

// setupPingPongHandlers sets up ping/pong handlers to maintain connection health
func (a *Agent) setupPingPongHandlers() {
	if a.Conn == nil {
		return
	}
	a.Conn.SetReadDeadline(time.Now().Add(pongWait))
	a.Conn.SetPingHandler(func(appData string) error {
		if a.Conn != nil {
			a.Conn.SetReadDeadline(time.Now().Add(pongWait))
			err := a.Conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(writeDeadline))
			if err != nil {
				logger.Log.Error("Failed to send pong", "err", err)
				return err
			}
		}
		return nil
	})
	a.Conn.SetPongHandler(func(appData string) error {
		if a.Conn != nil {
			a.Conn.SetReadDeadline(time.Now().Add(pongWait))
		}
		return nil
	})
}

// readPump handles incoming messages from master
func (a *Agent) readPump() {
	defer func() {
		logger.Log.Info("Read pump stopped")
	}()
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			if a.Conn == nil {
				return
			}
			a.Conn.SetReadDeadline(time.Now().Add(pongWait))
			msgType, msgBytes, err := a.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Log.Error("WebSocket read error", "err", err)
				}
				a.Close()
				return
			}
			switch msgType {
			case websocket.TextMessage:
				var msg models.Message
				if err := json.Unmarshal(msgBytes, &msg); err != nil {
					logger.Log.Warn("Unmarshalling error: TEXT", "err", err)
					continue
				}
				select {
				case a.incomingCh <- Outbound{Msg: &msg}:
				case <-a.ctx.Done():
					return
				default:
					// TODO: Handling backpressure
					logger.Log.Warn("Incoming channel full for %s, dropping message\n")
				}
			case websocket.BinaryMessage:
				select {
				case a.incomingCh <- Outbound{Binary: msgBytes}:
				case <-a.ctx.Done():
					return
				default:
					// TODO: Handling backpressure
					logger.Log.Warn("Incoming channel full for %s, dropping message\n")
				}
			default:
				logger.Log.Warn("Recieved mssg dropped. Neither TEXT/BINARY")
			}
			a.Conn.SetReadDeadline(time.Now().Add(pongWait))
		}
	}
}

// writePump handles outgoing messages to master
func (a *Agent) writePump() {
	defer func() {
		logger.Log.Info("Write pump stopped")
	}()
	for {
		select {
		case msg := <-a.sendCh:
			if a.Conn == nil {
				return
			}
			a.Conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if msg.Msg != nil {
				bytes, err := json.Marshal(*msg.Msg)
				if err != nil {
					logger.Log.Error("Marshalling error", "err", err)
					continue
				}
				if err := a.Conn.WriteMessage(websocket.TextMessage, bytes); err != nil {
					logger.Log.Error("Write error: TEXT", "err", err)
					a.Close()
					return
				}
			} else {
				if err := a.Conn.WriteMessage(websocket.BinaryMessage, msg.Binary); err != nil {
					logger.Log.Error("Write error: BINARY", "err", err)
					a.Close()
					return
				}
			}
		case <-a.ctx.Done():
			if a.Conn != nil {
				a.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, ""))
			}
			return
		}
	}
}

// dispatchPump dispatches incoming messages to registered handlers
func (a *Agent) dispatchPump() {
	defer func() {
		logger.Log.Info("Dispatch pump stopped")
	}()
	for {
		select {
		case msg := <-a.incomingCh:
			if msg.Msg != nil {
				messageRec := *msg.Msg

				// Debug: show raw message
				logger.Log.Info("Received message", "rawMsg", messageRec)

				// Debug: show type explicitly
				logger.Log.Info("Message type", "Type", messageRec.Type, "IsEmpty?", messageRec.Type == "")

				if handler, ok := a.Handlers[messageRec.Type]; ok {
					logger.Log.Debug("Found handler for message type", "type", messageRec.Type)
					if err := handler(&messageRec.Payload); err != nil {
						logger.Log.Error("Handler error", "type", messageRec.Type, "err", err)
					}
				} else {
					logger.Log.Warn("No handler for message type", "type", messageRec.Type, "payload", messageRec.Payload)
				}
			} else {
				// TODO: Recieving the tar bytes. Handle it
			}
		case <-a.ctx.Done():
			return
		}
	}
}

// RunPumps starts all pumps and sets up ping/pong handlers
func (a *Agent) RunPumps() {
	if a.Conn == nil {
		logger.Log.Error("Cannot start pumps: connection is nil")
		return
	}
	a.setupPingPongHandlers()
	go a.readPump()     // Reads messages
	go a.writePump()    // Writes messages
	go a.dispatchPump() // Dispatches to handlers
	logger.Log.Info("All pumps started")
}
