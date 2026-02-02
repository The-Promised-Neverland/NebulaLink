package ws

import (
	"context"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/gorilla/websocket"
)

type Connection struct {
	Name       string
	Id         string
	Conn       *websocket.Conn
	OS         string
	LastSeen   time.Time
	SendCh     chan models.Message
	IncomingCh chan models.Message
	Ctx        context.Context
	Cancel     context.CancelFunc
}

func NewConnection(name string, id string, conn *websocket.Conn) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		Name:       name,
		Conn:       conn,
		Id:         id,
		LastSeen:   time.Now(),
		SendCh:     make(chan models.Message, 100),
		IncomingCh: make(chan models.Message, 500),
		Ctx:        ctx,
		Cancel:     cancel,
	}
}
