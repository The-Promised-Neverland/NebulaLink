package ws

import (
	"context"
	"sync"
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
	StreamCh   chan []byte
	RelayTo	   string  	// ID of the file system requesting agent. Value set up by read pump
	Ctx        context.Context
	Cancel     context.CancelFunc
	wg         sync.WaitGroup
	connMutex  sync.RWMutex
}

func NewConnection(name string, id string, os string, conn *websocket.Conn) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		Name:       name,
		Id:         id,
		Conn:       conn,
		OS:         os,
		LastSeen:   time.Now(),
		SendCh:     make(chan models.Message, 100),
		IncomingCh: make(chan models.Message, 500),
		StreamCh:   make(chan []byte, 1024*64),
		Ctx:        ctx,
		Cancel:     cancel,
	}
}
