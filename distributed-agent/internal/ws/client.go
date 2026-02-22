package ws

import (
	"context"
	"errors"
	"os"
	"runtime"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/internal/p2p"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/The-Promised-Neverland/agent/pkg/utils"
	"github.com/gorilla/websocket"
)

type Outbound struct {
	Msg    *models.Message
	Binary []byte
}

type Agent struct {
	Conn         *websocket.Conn
	Config       *config.Config
	Handlers     map[string]func(msg *any) error
	sendCh       chan Outbound
	incomingCh   chan Outbound
	ctx          context.Context
	cancel       context.CancelFunc
	TempFile     *os.File
	tempFilePath string
	sourceAgent  string
	P2PClient    *p2p.P2PClient
}

func NewAgent(cfg *config.Config, parentCtx context.Context) *Agent {
	ctx, cancel := context.WithCancel(parentCtx)
	agent := &Agent{
		Config:     cfg,
		Handlers:   make(map[string]func(msg *any) error),
		sendCh:     make(chan Outbound, 256),
		incomingCh: make(chan Outbound, 256),
		ctx:        ctx,
		cancel:     cancel,
	}
	agent.P2PClient = p2p.NewP2PClient(cfg.AgentID(), cfg, func(msg interface{}) error {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			return errors.New("invalid message format")
		}
		msgType, _ := msgMap["type"].(string)
		delete(msgMap, "type")
		message := models.Message{
			Type:    msgType,
			Payload: msgMap,
		}
		return agent.Send(Outbound{Msg: &message})
	})

	return agent
}

func (a *Agent) AgentDisconnected() <-chan struct{} {
	return a.ctx.Done()
}

func (a *Agent) RegisterHandler(msgType string, handler func(msg *any) error) {
	a.Handlers[msgType] = handler
}

func (a *Agent) Connect() error {
	baseURL := a.Config.MasterServerConn()
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macOS"
	} else if osName == "windows" {
		osName = "Windows"
	} else if osName == "linux" {
		osName = "Linux"
	}
	wsURL := utils.BuildWebSocketURL(baseURL, a.Config.AgentID(), a.Config.AgentName(), osName)
	logger.Log.Info("Attempting connection", "url", wsURL)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		logger.Log.Error("Connection error", "err", err)
		return err
	}
	a.Conn = conn
	logger.Log.Info("Connected to master", "url", wsURL)
	return nil
}

func (a *Agent) Send(out Outbound) error {
	select {
	case <-a.ctx.Done():
		return errors.New("connection is closed")
	case a.sendCh <- out:
		return nil
	default:
		logger.Log.Warn("Send buffer full, dropping message")
		return errors.New("send buffer full")
	}
}

func (a *Agent) Close() error {
	if a.TempFile != nil {
		a.TempFile.Close()
		if a.tempFilePath != "" {
			os.Remove(a.tempFilePath)
		}
		a.TempFile = nil
		a.tempFilePath = ""
		a.sourceAgent = ""
	}
	if a.P2PClient != nil {
		a.P2PClient.CloseConnection("")
	}
	a.cancel()
	if a.Conn != nil {
		return a.Conn.Close()
	}
	return nil
}
