package ws

import (
	"errors"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/The-Promised-Neverland/agent/pkg/utils"
	"github.com/gorilla/websocket"
)

type Agent struct {
	Conn         *websocket.Conn
	Config       *config.Config
	Handlers     map[string]func(msg *any) error
	sendCh       chan models.Message
	incomingCh   chan models.Message
	disconnectCh chan struct{}
	isClosed     bool
}

func NewAgent(cfg *config.Config) *Agent {
	return &Agent{
		Config:       cfg,
		Handlers:     make(map[string]func(msg *any) error),
		sendCh:       make(chan models.Message, 256),
		incomingCh:   make(chan models.Message, 256),
		disconnectCh: make(chan struct{}),
	}
}

func (a *Agent) AgentDisconnected() <-chan struct{} {
	return a.disconnectCh
}

func (a *Agent) RegisterHandler(msgType string, handler func(msg *any) error) {
	a.Handlers[msgType] = handler
}

func (a *Agent) Connect() error {
	baseURL := a.Config.MasterServerConn()
	wsURL := utils.BuildWebSocketURL(baseURL, a.Config.AgentID())
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	logger.Log.Info("🔄 Attempting connection", "url", wsURL)
	if err != nil {
		logger.Log.Error("❌ Connection error", "err", err)
		return err
	}
	a.Conn = conn
	logger.Log.Info("✅ Connected to master", "url", wsURL)
	return nil
}

func (a *Agent) Send(msg models.Message) error {
	if a.isClosed {
		return errors.New("connection is closed")
	}
	select {
	case a.sendCh <- msg:
		return nil
	default:
		logger.Log.Warn("⚠️ Send buffer full, dropping message")
		return errors.New("send buffer full")
	}
}

// Close safely closes the connection and stops pumps
func (a *Agent) Close() error {
	if a.isClosed {
		return nil
	}
	a.isClosed = true
	close(a.disconnectCh)
	return a.Conn.Close()
}
