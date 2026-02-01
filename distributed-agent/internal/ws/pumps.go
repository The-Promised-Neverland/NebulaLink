package ws

import (
	"encoding/json"
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
		logger.Log.Info("Received ping from master")
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
		logger.Log.Info("Received pong from master")
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
		}
		if a.Conn == nil {
			return
		}
		a.Conn.SetReadDeadline(time.Now().Add(pongWait))
		_, msgBytes, err := a.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log.Error("WebSocket read error", "err", err)
			}
			a.Close()
			return
		}
		a.Conn.SetReadDeadline(time.Now().Add(pongWait))
		var msg models.Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			logger.Log.Warn("Failed to parse message", "err", err)
			continue
		}
		select {
		case a.incomingCh <- msg:
		case <-a.ctx.Done():
			return
		default:
			logger.Log.Warn("Incoming buffer full, dropping message")
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
			if err := a.Conn.WriteJSON(msg); err != nil {
				logger.Log.Error("Write error", "err", err)
				a.Close()
				return
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
			if handler, ok := a.Handlers[msg.Type]; ok {
				if err := handler(&msg.Payload); err != nil {
					logger.Log.Error("Handler error", "type", msg.Type, "err", err)
				}
			} else {
				logger.Log.Warn("No handler for message type", "type", msg.Type)
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
	go a.readPump()      // Reads messages
	go a.writePump()     // Writes messages
	go a.dispatchPump()  // Dispatches to handlers
	logger.Log.Info("All pumps started")
}
