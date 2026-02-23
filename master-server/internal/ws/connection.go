package ws

import (
	"context"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/transfer"
	"github.com/gorilla/websocket"
)

type Connection struct {
	Name           string
	Id             string
	Conn           *websocket.Conn
	OS             string
	LastSeen       time.Time
	SendCh         chan transfer.Outbound
	IncomingCh     chan transfer.Outbound
	StreamCh       chan []byte
	RelayTo        string
	Ctx            context.Context
	Cancel         context.CancelFunc
	wg             sync.WaitGroup
	ConnMutex      sync.RWMutex
	PublicEndpoint string
}

func (c *Connection) GetPublicEndpoint() string {
	c.ConnMutex.RLock()
	defer c.ConnMutex.RUnlock()
	return c.PublicEndpoint
}

func (c *Connection) SetRelayTo(agentID string) {
	c.ConnMutex.Lock()
	defer c.ConnMutex.Unlock()
	c.RelayTo = agentID
}

func NewConnection(name string, id string, os string, conn *websocket.Conn) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		Name:           name,
		Id:             id,
		Conn:           conn,
		OS:             os,
		LastSeen:       time.Now(),
		SendCh:         make(chan transfer.Outbound, 1024*64),
		IncomingCh:     make(chan transfer.Outbound, 1024*64),
		StreamCh:       make(chan []byte, 1024*64),
		Ctx:            ctx,
		Cancel:         cancel,
		PublicEndpoint: "",
	}
}
