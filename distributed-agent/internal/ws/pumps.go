package ws

import (
	"encoding/json"
	"time"

	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

const (
	readDeadline = 70 * time.Second
)

// connectionMonitor checks if we're receiving pings from master
func (a *Agent) connectionMonitor() {
    a.Conn.SetReadDeadline(time.Now().Add(readDeadline))
    a.Conn.SetPingHandler(func(string) error {
        logger.Log.Info("🏓 Received ping from master")
        a.Conn.SetReadDeadline(time.Now().Add(readDeadline))
        return nil
    })
}

// readPump handles incoming messages from master
func (a *Agent) readPump() {
	defer func() {
		logger.Log.Info("🔴 Read pump stopped")
	}()
	for {
		_, msgBytes, err := a.Conn.ReadMessage()
		if err != nil {
			a.Close()
			return
		}
		a.Conn.SetReadDeadline(time.Now().Add(readDeadline))
		var msg models.Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			logger.Log.Warn("⚠️ Failed to parse message", "warn", err)
			continue
		}
		select {
		case a.incomingCh <- msg:
		case <-a.disconnectCh:
			return
		default:
			logger.Log.Warn("⚠️ Incoming buffer full, dropping message")
		}
	}
}

// writePump handles outgoing messages to master
func (a *Agent) writePump() {
	defer func() {
		logger.Log.Info("🔴 Write pump stopped")
	}()
	for {
		select {
		case msg := <-a.sendCh:
			if err := a.Conn.WriteJSON(msg); err != nil {
				a.Close()
				return
			}
		case <-a.disconnectCh:
			return
		}
	}
}

// dispatchPump dispatches incoming messages to registered handlers
func (a *Agent) dispatchPump() {
	defer func() {
		logger.Log.Info("🔴 Dispatch pump stopped")
	}()
	for {
		select {
		case msg := <-a.incomingCh:
			if handler, ok := a.Handlers[msg.Type]; ok {
				if err := handler(&msg.Payload); err != nil {
					logger.Log.Error("❌ Handler error", "type", msg.Type, "err", err)
				}
			} else {
				logger.Log.Warn("⚠️ No handler for message type", "type", msg.Type)
			}
		case <-a.disconnectCh:
			return
		}
	}
}

// RunPumps starts all pumps and connection monitor
func (a *Agent) RunPumps() {
	go a.connectionMonitor() // Monitors ping health
	go a.readPump()          // Reads messages
	go a.writePump()         // Writes messages
	go a.dispatchPump()      // Dispatches to handlers
	logger.Log.Info("✅ All pumps started")
}
