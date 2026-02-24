package transfer

import (
	"fmt"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/internal/ws"
)

type TransferManager struct {
	config          *config.Config
	p2pClient       *P2PClient
	businessService *service.Service
	agent           *ws.Agent
	extractor       Extractor
	currentTransfer Transferer
	ctx             *TransferContext
}

// NewTransferManager creates a new transfer manager and initializes P2P client
func NewTransferManager(cfg *config.Config, businessService *service.Service, agent *ws.Agent) *TransferManager {
	extractor := NewTarExtractor(cfg)
	ctx := &TransferContext{
		Mode: ModeRelay,
	}
	p2pClient := NewP2PClient(cfg.AgentID(), cfg, func(msg *models.Message) error {
		return agent.Send(ws.Outbound{Msg: msg})
	})
	return &TransferManager{
		config:          cfg,
		p2pClient:       p2pClient,
		businessService: businessService,
		agent:           agent,
		extractor:       extractor,
		ctx:             ctx,
	}
}

func (m *TransferManager) GetTransferer(mode string) (Transferer, error) {
	transferMode := TransferMode(mode)
	if transferMode == "" {
		return nil, fmt.Errorf("transfer mode not specified - master must coordinate first")
	}
	m.ctx.Mode = transferMode
	switch transferMode {
	case ModeP2P:
		if m.p2pClient == nil {
			return nil, fmt.Errorf("P2P client not initialized")
		}
		m.currentTransfer = NewP2PTransfer(m.ctx, m.p2pClient, m.config, m.businessService, m.agent, m.extractor)
		return m.currentTransfer, nil
	case ModeRelay:
		m.currentTransfer = NewRelayTransfer(m.ctx, m.config, m.businessService, m.agent, m.extractor)
		return m.currentTransfer, nil
	// Add more modes. TURN etc
	default:
		return nil, fmt.Errorf("unknown transfer mode: %s", mode)
	}
}

func (m *TransferManager) Send(path string, requestingAgentID string, mode string) error {
	transferer, err := m.GetTransferer(mode)
	if err != nil {
		return fmt.Errorf("failed to get transferer: %w", err)
	}
	return transferer.Send(path, requestingAgentID)
}

func (m *TransferManager) Receive(sourceAgentID string, mode string) error {
	transferer, err := m.GetTransferer(mode)
	if err != nil {
		return fmt.Errorf("failed to get transferer: %w", err)
	}

	return transferer.Receive(sourceAgentID)
}

func (m *TransferManager) WriteChunk(chunk []byte) error {
	if m.currentTransfer == nil {
		return fmt.Errorf("no active transfer")
	}
	return m.currentTransfer.WriteChunk(chunk)
}

func (m *TransferManager) Complete() error {
	if m.currentTransfer == nil {
		return fmt.Errorf("no active transfer to complete")
	}
	return m.currentTransfer.Complete()
}

func (m *TransferManager) GetContext() *TransferContext {
	return m.ctx
}

func (m *TransferManager) SetConnectionID(connectionID string) {
	m.ctx.ConnectionID = connectionID
}

// AttemptP2PConnection attempts a P2P connection
func (m *TransferManager) AttemptP2PConnection(connectionID, targetAgentID, targetEndpoint string, attemptNumber, countdownSeconds int) error {
	if m.p2pClient == nil {
		return fmt.Errorf("P2P client not initialized")
	}
	return m.p2pClient.AttemptConnection(connectionID, targetAgentID, targetEndpoint, attemptNumber, countdownSeconds)
}

// CloseP2PConnection closes a P2P connection
func (m *TransferManager) CloseP2PConnection(connectionID string) {
	if m.p2pClient != nil {
		m.p2pClient.CloseConnection(connectionID)
	}
}

// CloseAll closes all P2P connections and cleans up
func (m *TransferManager) CloseAll() {
	if m.p2pClient != nil {
		m.p2pClient.CloseConnection("")
	}
}
