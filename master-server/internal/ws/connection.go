package ws

import (
	"context"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/gorilla/websocket"
)

type Connection struct {
	Role         string
	Conn         *websocket.Conn
	OS           string
	LastSeen     time.Time
	SendCh       chan models.Message
	IncomingCh   chan models.Message
	Ctx         context.Context
    Cancel      context.CancelFunc
}

func NewConnection(role string, conn *websocket.Conn) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		Conn:         conn,
		Role:         role,
		LastSeen:     time.Now(),
		SendCh:       make(chan models.Message, 100),
		IncomingCh:   make(chan models.Message, 500),
		Ctx:        ctx,
        Cancel:     cancel,
	}
}
