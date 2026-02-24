package transfer

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/internal/ws"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

const (
	P2PConnectionTimeout = 10 * time.Second
)

type P2PConnection struct {
	ConnectionID   string
	TargetAgentID  string
	TargetEndpoint string
	Conn           net.Conn
	Status         string // "connecting", "connected", "failed", "closed"
	AttemptNumber  int
	Mu             sync.RWMutex
}

type P2PClient struct {
	agentID    string
	config     *config.Config
	activeConn *P2PConnection
	mu         sync.RWMutex
	sendFunc   func(msg *models.Message) error
}

func NewP2PClient(agentID string, cfg *config.Config, sendFunc func(msg *models.Message) error) *P2PClient {
	return &P2PClient{
		agentID:  agentID,
		config:   cfg,
		sendFunc: sendFunc,
	}
}

// AttemptConnection attempts TCP hole punching connection
func (p *P2PClient) AttemptConnection(connectionID, targetAgentID, targetEndpoint string, attemptNumber int, countdownSeconds int) error {
	p.mu.Lock()
	if p.activeConn != nil && p.activeConn.ConnectionID != connectionID {
		p.activeConn.Mu.Lock()
		if p.activeConn.Conn != nil {
			p.activeConn.Conn.Close()
		}
		p.activeConn.Mu.Unlock()
	}
	conn := &P2PConnection{
		ConnectionID:   connectionID,
		TargetAgentID:  targetAgentID,
		TargetEndpoint: targetEndpoint,
		Status:         "connecting",
		AttemptNumber:  attemptNumber,
	}
	p.activeConn = conn
	p.mu.Unlock()
	time.Sleep(time.Duration(countdownSeconds) * time.Second)
	conn.Mu.Lock()
	conn.Status = "connecting"
	conn.Mu.Unlock()
	logger.Log.Info("[P2P] Attempting P2P connection...", "connection_id", connectionID, "target", targetEndpoint, "attempt", attemptNumber, "countdown", countdownSeconds)
	tcpConn, err := net.DialTimeout("tcp", targetEndpoint, P2PConnectionTimeout)
	if err != nil {
		conn.Mu.Lock()
		conn.Status = "failed"
		conn.Mu.Unlock()
		logger.Log.Warn("[P2P] P2P connection failed", "connection_id", connectionID, "target", targetEndpoint, "error", err)
		p.reportFailure(connectionID, fmt.Sprintf("connection failed: %v", err))
		p.mu.Lock()
		if p.activeConn != nil && p.activeConn.ConnectionID == connectionID {
			p.activeConn = nil
		}
		p.mu.Unlock()
		return fmt.Errorf("failed to connect: %w", err)
	}
	conn.Mu.Lock()
	conn.Conn = tcpConn
	conn.Status = "connected"
	conn.Mu.Unlock()
	logger.Log.Info("[P2P] SUCCESS: P2P connection established", "connection_id", connectionID, "target", targetEndpoint)
	p.reportSuccess(connectionID)
	return nil
}

// SendFileOverP2P sends file over established P2P connection
func (p *P2PClient) SendFileOverP2P(connectionID string, fileReader io.Reader) error {
	p.mu.RLock()
	conn := p.activeConn
	p.mu.RUnlock()
	if conn == nil || conn.ConnectionID != connectionID {
		return fmt.Errorf("no active P2P connection for %s", connectionID)
	}
	conn.Mu.RLock()
	tcpConn := conn.Conn
	status := conn.Status
	conn.Mu.RUnlock()
	if status != "connected" || tcpConn == nil {
		return fmt.Errorf("P2P connection not ready: %s", status)
	}
	logger.Log.Info("[P2P] Starting P2P file transfer (sending bytes)", "connection_id", connectionID)
	written, err := io.Copy(tcpConn, fileReader)
	if err != nil {
		logger.Log.Error("[P2P] P2P file transfer FAILED", "connection_id", connectionID, "error", err, "bytes_written", written)
		p.CloseConnection(connectionID) // Close connection on error
		return fmt.Errorf("failed to send file: %w", err)
	}
	logger.Log.Info("[P2P] P2P file transfer SUCCESS: completed sending", "connection_id", connectionID, "bytes_sent", written)
	p.CloseConnection(connectionID) // Close connection on success
	return nil
}

// ReceiveFileOverP2P receives file over P2P connection
func (p *P2PClient) ReceiveFileOverP2P(connectionID string, fileWriter io.Writer) error {
	p.mu.RLock()
	conn := p.activeConn
	p.mu.RUnlock()
	if conn == nil || conn.ConnectionID != connectionID {
		return fmt.Errorf("no active P2P connection for %s", connectionID)
	}
	conn.Mu.RLock()
	tcpConn := conn.Conn
	status := conn.Status
	conn.Mu.RUnlock()
	if status != "connected" || tcpConn == nil {
		return fmt.Errorf("P2P connection not ready: %s", status)
	}
	logger.Log.Info("[P2P] Starting P2P file receive (receiving bytes)", "connection_id", connectionID)
	received, err := io.Copy(fileWriter, tcpConn)
	if err != nil && err != io.EOF {
		logger.Log.Error("[P2P] P2P file receive FAILED", "connection_id", connectionID, "error", err, "bytes_received", received)
		p.CloseConnection(connectionID) // Close connection on error
		return fmt.Errorf("failed to receive file: %w", err)
	}
	logger.Log.Info("[P2P] P2P file receive SUCCESS: completed receiving", "connection_id", connectionID, "bytes_received", received)
	p.CloseConnection(connectionID) // Close connection on success
	return nil
}

// CloseConnection closes the P2P connection
func (p *P2PClient) CloseConnection(connectionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.activeConn == nil {
		return
	}
	if connectionID != "" && p.activeConn.ConnectionID != connectionID {
		return
	}
	p.activeConn.Mu.Lock()
	if p.activeConn.Conn != nil {
		p.activeConn.Conn.Close()
	}
	p.activeConn.Status = "closed"
	closedID := p.activeConn.ConnectionID
	p.activeConn.Mu.Unlock()
	p.activeConn = nil
	logger.Log.Info("P2P connection closed", "connection_id", closedID)
}

// GetActiveConnection returns the active P2P connection by connection ID
func (p *P2PClient) GetActiveConnection(connectionID string) *P2PConnection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.activeConn != nil && p.activeConn.ConnectionID == connectionID {
		return p.activeConn
	}
	return nil
}

// GetActiveConnectionByTarget returns the active P2P connection by target agent ID
func (p *P2PClient) GetActiveConnectionByTarget(targetAgentID string) *P2PConnection {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.activeConn != nil && p.activeConn.TargetAgentID == targetAgentID {
		return p.activeConn
	}
	return nil
}

// HasActiveConnection checks if there's any active P2P connection
func (p *P2PClient) HasActiveConnection() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeConn != nil && p.activeConn.Status == "connected"
}

// reportSuccess reports P2P connection success to master using transfer status
func (p *P2PClient) reportSuccess(connectionID string) {
	if p.sendFunc == nil {
		return
	}
	msg := &models.Message{
		Type: "master_transfer_status",
		Payload: map[string]interface{}{
			"status":        "p2p_success",
			"connection_id": connectionID,
			"agent_id":      p.agentID,
		},
	}
	if err := p.sendFunc(msg); err != nil {
		logger.Log.Error("Failed to report P2P success", "error", err)
	}
}

// reportFailure reports P2P connection failure to master using transfer status
func (p *P2PClient) reportFailure(connectionID, reason string) {
	if p.sendFunc == nil {
		return
	}
	msg := &models.Message{
		Type: "master_transfer_status",
		Payload: map[string]interface{}{
			"status":        "p2p_failed",
			"connection_id": connectionID,
			"reason":        reason,
			"agent_id":      p.agentID,
		},
	}
	if err := p.sendFunc(msg); err != nil {
		logger.Log.Error("Failed to report P2P failure", "error", err)
	}
}

// WaitForConnection waits for P2P connection to be established
func (p *P2PClient) WaitForConnection(connectionID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		p.mu.RLock()
		conn := p.activeConn
		p.mu.RUnlock()
		if conn != nil && conn.ConnectionID == connectionID {
			conn.Mu.RLock()
			status := conn.Status
			conn.Mu.RUnlock()
			if status == "connected" {
				return nil
			}
			if status == "failed" {
				return fmt.Errorf("P2P connection failed")
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("P2P connection timeout")
}

type P2PTransfer struct {
	ctx             *TransferContext
	p2pClient       *P2PClient
	config          *config.Config
	businessService *service.Service
	agent           *ws.Agent
	extractor       Extractor
}

func NewP2PTransfer(ctx *TransferContext, p2pClient *P2PClient, cfg *config.Config, businessService *service.Service, agent *ws.Agent, extractor Extractor) *P2PTransfer {
	return &P2PTransfer{
		ctx:             ctx,
		p2pClient:       p2pClient,
		config:          cfg,
		businessService: businessService,
		agent:           agent,
		extractor:       extractor,
	}
}

func (p *P2PTransfer) GetMode() TransferMode {
	return ModeP2P
}

func (p *P2PTransfer) Send(path string, requestingAgentID string) error {
	logger.Log.Info("[P2P] Starting P2P transfer", "path", path, "target", requestingAgentID)
	p2pConn := p.p2pClient.GetActiveConnectionByTarget(requestingAgentID)
	if p2pConn == nil || p2pConn.Status != "connected" {
		status := "no_connection"
		if p2pConn != nil {
			status = p2pConn.Status
		}
		logger.Log.Error("[P2P] P2P connection not available", "status", status, "target", requestingAgentID)
		return fmt.Errorf("P2P connection not available: status=%v", status)
	}
	logger.Log.Info("[P2P] P2P connection available, starting file transfer", "connection_id", p2pConn.ConnectionID, "target", requestingAgentID, "path", path)
	dataCh, errCh := p.businessService.StreamRequestedFileSystem(path)
	reader := &channelReader{dataCh: dataCh, errCh: errCh}
	if err := p.p2pClient.SendFileOverP2P(p2pConn.ConnectionID, reader); err != nil {
		return fmt.Errorf("P2P send failed: %w", err)
	}
	logger.Log.Info("[P2P] P2P transfer completed successfully, reporting to master")
	doneMsg := models.Message{
		Type: models.MasterMsgTransferStatus,
		Payload: map[string]interface{}{
			"status":   "completed",
			"agent_id": p.config.AgentID(),
		},
	}
	p.agent.Send(ws.Outbound{Msg: &doneMsg})
	logger.Log.Info("[P2P] Reported 'completed' status to master")
	return nil
}

func (p *P2PTransfer) Receive(sourceAgentID string) error {
	logger.Log.Info("[P2P] Preparing to receive P2P transfer", "sourceAgent", sourceAgentID)
	if err := p.createTempFile(sourceAgentID); err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	time.Sleep(1 * time.Second) // give some time for file creation. OS delay expected
	p2pConn := p.p2pClient.GetActiveConnectionByTarget(sourceAgentID)
	if p2pConn == nil || p2pConn.Status != "connected" {
		logger.Log.Error("[P2P] P2P connection not available for receiving", "sourceAgent", sourceAgentID)
		return fmt.Errorf("P2P connection not available for receiving")
	}
	logger.Log.Info("[P2P] P2P connection ready, starting to receive file", "connection_id", p2pConn.ConnectionID, "sourceAgent", sourceAgentID)
	if err := p.p2pClient.ReceiveFileOverP2P(p2pConn.ConnectionID, p.ctx.TempFile); err != nil {
		return fmt.Errorf("P2P receive failed: %w", err)
	}
	logger.Log.Info("P2P file received successfully, waiting for master to send completed status")
	return nil
}

func (p *P2PTransfer) WriteChunk(chunk []byte) error {
	return fmt.Errorf("WriteChunk not supported in P2P mode. Data is sent over TCP directly to reciever")
}

func (p *P2PTransfer) Complete() error {
	return p.completeTransfer()
}

func (p *P2PTransfer) createTempFile(sourceAgentID string) error {
	if p.ctx.TempFile != nil {
		p.ctx.TempFile.Close()
		os.Remove(p.ctx.TempFilePath)
	}
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "transfer_*.tar")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	p.ctx.TempFile = tempFile
	p.ctx.TempFilePath = tempFile.Name()
	p.ctx.SourceAgentID = sourceAgentID
	logger.Log.Info("Created temp file for P2P transfer", "sourceAgent", sourceAgentID, "tempFile", p.ctx.TempFilePath)
	return nil
}

func (p *P2PTransfer) completeTransfer() error {
	if p.ctx.TempFile == nil {
		return fmt.Errorf("no active transfer to complete")
	}
	if err := p.ctx.TempFile.Close(); err != nil {
		logger.Log.Error("Failed to close temp file", "err", err)
	}
	tempPath := p.ctx.TempFilePath
	sourceAgent := p.ctx.SourceAgentID
	p.ctx.TempFile = nil
	p.ctx.TempFilePath = ""
	p.ctx.SourceAgentID = ""
	if err := p.extractor.ExtractTar(tempPath, sourceAgent); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to extract tar: %w", err)
	}
	if err := os.Remove(tempPath); err != nil {
		logger.Log.Warn("Failed to remove temp file", "path", tempPath, "err", err)
	}
	logger.Log.Info("P2P transfer completed and file extracted", "sourceAgent", sourceAgent)
	return nil
}
