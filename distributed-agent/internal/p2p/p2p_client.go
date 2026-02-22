package p2p

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

const (
	ConnectionTimeout = 10 * time.Second
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
	sendFunc   func(msg interface{}) error
}

func NewP2PClient(agentID string, cfg *config.Config, sendFunc func(msg interface{}) error) *P2PClient {
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
	logger.Log.Info("P2P connection attempt",
		"connection_id", connectionID,
		"target", targetEndpoint,
		"attempt", attemptNumber,
		"countdown", countdownSeconds)
	time.Sleep(time.Duration(countdownSeconds) * time.Second)
	conn.Mu.Lock()
	conn.Status = "connecting"
	conn.Mu.Unlock()
	tcpConn, err := net.DialTimeout("tcp", targetEndpoint, ConnectionTimeout)
	if err != nil {
		conn.Mu.Lock()
		conn.Status = "failed"
		conn.Mu.Unlock()
		logger.Log.Warn("P2P connection failed",
			"connection_id", connectionID,
			"target", targetEndpoint,
			"error", err)
		p.reportFailure(connectionID, fmt.Sprintf("connection failed: %v", err))
		return fmt.Errorf("failed to connect: %w", err)
	}
	conn.Mu.Lock()
	conn.Conn = tcpConn
	conn.Status = "connected"
	conn.Mu.Unlock()
	logger.Log.Info("P2P connection established",
		"connection_id", connectionID,
		"target", targetEndpoint)
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
	logger.Log.Info("Starting P2P file transfer", "connection_id", connectionID)
	written, err := io.Copy(tcpConn, fileReader)
	if err != nil {
		logger.Log.Error("P2P file transfer error", "error", err, "bytes_written", written)
		return fmt.Errorf("failed to send file: %w", err)
	}
	logger.Log.Info("P2P file transfer completed", "connection_id", connectionID, "bytes", written)
	p.CloseConnection(connectionID)
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
	logger.Log.Info("Starting P2P file receive", "connection_id", connectionID)
	received, err := io.Copy(fileWriter, tcpConn)
	if err != nil && err != io.EOF {
		logger.Log.Error("P2P file receive error", "error", err, "bytes_received", received)
		return fmt.Errorf("failed to receive file: %w", err)
	}
	logger.Log.Info("P2P file receive completed", "connection_id", connectionID, "bytes", received)
	p.CloseConnection(connectionID)
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

// reportSuccess reports P2P connection success to master
func (p *P2PClient) reportSuccess(connectionID string) {
	if p.sendFunc == nil {
		return
	}
	msg := map[string]interface{}{
		"type":          "agent_p2p_success",
		"connection_id": connectionID,
	}
	if err := p.sendFunc(msg); err != nil {
		logger.Log.Error("Failed to report P2P success", "error", err)
	}
}

// reportFailure reports P2P connection failure to master
func (p *P2PClient) reportFailure(connectionID, reason string) {
	if p.sendFunc == nil {
		return
	}
	msg := map[string]interface{}{
		"type":          "agent_p2p_failed",
		"connection_id": connectionID,
		"reason":        reason,
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
