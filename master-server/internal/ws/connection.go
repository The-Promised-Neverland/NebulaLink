package ws

import (
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/gorilla/websocket"
)

type Connection struct {
	Role         string
	Conn         *websocket.Conn
	OS           string
	LastSeen     time.Time
	DisconnectCh chan struct{}
	SendCh       chan models.Message
	IncomingCh   chan models.Message
}

func NewConnection(role string, conn *websocket.Conn) *Connection {
	return &Connection{
		Conn:         conn,
		Role:         role,
		LastSeen:     time.Now(),
		DisconnectCh: make(chan struct{}),
		SendCh:       make(chan models.Message, 100),
		IncomingCh:   make(chan models.Message, 500),
	}
}
