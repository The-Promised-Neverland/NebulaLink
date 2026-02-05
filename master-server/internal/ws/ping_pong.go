package ws

import (
	"time"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
	writeWait  = 10 * time.Second
)

func (h *WSHub) handlePong(agent *Connection) {
	agent.connMutex.RLock()
	if agent.Conn == nil {
		agent.connMutex.RUnlock()
		return
	}
	conn := agent.Conn
	agent.connMutex.RUnlock()

	conn.SetPongHandler(func(string) error {
		agent.connMutex.RLock()
		if agent.Conn != nil {
			agent.Conn.SetReadDeadline(time.Now().Add(pongWait))
		}
		agent.connMutex.RUnlock()
		h.Mutex.Lock()
		agent.LastSeen = time.Now()
		h.Mutex.Unlock()
		return nil
	})
}
