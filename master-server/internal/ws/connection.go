package ws

import (
	"context"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/gorilla/websocket"
)

type Outbound struct {
	Msg    *models.Message
	Binary []byte
}

type Connection struct {
	Name           string
	Id             string
	Conn           *websocket.Conn
	OS             string
	LastSeen       time.Time
	SendCh         chan Outbound
	IncomingCh     chan Outbound
	StreamCh       chan []byte
	RelayTo        string // ID of the file system requesting agent. Value set up by read pump
	InitiatedSent  bool   // Track if we've already sent "initiated" message to avoid duplicates
	Ctx            context.Context
	Cancel         context.CancelFunc
	wg             sync.WaitGroup
	connMutex      sync.RWMutex
	PublicEndpoint string
}

func NewConnection(name string, id string, os string, conn *websocket.Conn) *Connection {
	ctx, cancel := context.WithCancel(context.Background())
	return &Connection{
		Name:       name,
		Id:         id,
		Conn:       conn,
		OS:         os,
		LastSeen:   time.Now(),
		SendCh:     make(chan Outbound, 1024*64),
		IncomingCh: make(chan Outbound, 1024*64),
		StreamCh:   make(chan []byte, 1024*64),
		Ctx:        ctx,
		Cancel:     cancel,
		PublicEndpoint: "",
	}
}
