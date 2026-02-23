package ws

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/google/uuid"
)

const (
	MaxRetries        = 3
	ConnectionTimeout = 10 * time.Second
	InitialBackoff    = 1 * time.Second
)

type TransferState struct {
	ConnectionID    string
	RequestingAgent string
	SourceAgent     string
	AttemptNumber   int
	RetryCount      int
	MaxRetries      int
	Status          string // "attempting", "success", "relay", "failed"
	StartTime       time.Time
	LastAttemptTime time.Time
	CancelFunc      context.CancelFunc
	successCh       chan bool
	failureCh       chan error
	mu              sync.RWMutex
}

type P2PManager struct {
	activeTransfers map[string]*TransferState
	mu              sync.RWMutex
	wshub           *WSHub
}

func NewP2PManager(wshub *WSHub) *P2PManager {
	return &P2PManager{
		activeTransfers: make(map[string]*TransferState),
		wshub:           wshub,
	}
}

// StartP2PTransfer initiates a P2P transfer with retry management
func (p *P2PManager) StartTransfer(requestingAgent, sourceAgent string) (string, error) {
	trxf_connection_uuid := uuid.New().String()
	requestingEndpoint, err := p.getAgentEndpoint(requestingAgent)
	if err != nil {
		return "", fmt.Errorf("failed to get requesting agent endpoint: %w", err)
	}
	sourceEndpoint, err := p.getAgentEndpoint(sourceAgent)
	if err != nil {
		return "", fmt.Errorf("failed to get source agent endpoint: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	state := &TransferState{
		ConnectionID:    trxf_connection_uuid,
		RequestingAgent: requestingAgent,
		SourceAgent:     sourceAgent,
		AttemptNumber:   1,
		RetryCount:      0,
		MaxRetries:      MaxRetries,
		Status:          "attempting",
		StartTime:       time.Now(),
		LastAttemptTime: time.Now(),
		CancelFunc:      cancel,
		successCh:       make(chan bool, 1),
		failureCh:       make(chan error, 1),
	}
	p.mu.Lock()
	p.activeTransfers[trxf_connection_uuid] = state
	p.mu.Unlock()
	go p.manageRetries(ctx, trxf_connection_uuid, requestingEndpoint, sourceEndpoint)
	return trxf_connection_uuid, nil
}

// manageRetries handles retry logic with exponential backoff
func (p *P2PManager) manageRetries(ctx context.Context, trxf_connection_uuid, requestingEndpoint, sourceEndpoint string) {
	defer func() {
		p.mu.Lock()
		delete(p.activeTransfers, trxf_connection_uuid)
		p.mu.Unlock()
	}()
	p.mu.RLock()
	state := p.activeTransfers[trxf_connection_uuid]
	p.mu.RUnlock()
	if state == nil {
		return
	}
	attemptNum := 1
	retryCount := 0
	for retryCount < MaxRetries {
		err := p.sendP2PInitiation(trxf_connection_uuid, state.RequestingAgent, state.SourceAgent, requestingEndpoint, sourceEndpoint, attemptNum)
		if err != nil {
			fmt.Printf("Failed to send P2P initiation for %s: %v\n", trxf_connection_uuid, err)
			retryCount++
			attemptNum++
			continue
		}
		backoffDelay := time.Duration(math.Pow(2, float64(retryCount))) * InitialBackoff
		select {
		case <-time.After(backoffDelay):
			select {
			case success := <-state.successCh:
				if success {
					state.mu.Lock()
					state.Status = "success"
					state.mu.Unlock()
					fmt.Printf("P2P transfer %s succeeded on attempt %d\n", trxf_connection_uuid, attemptNum)
					return
				}
			case err := <-state.failureCh:
				fmt.Printf("P2P transfer %s failed on attempt %d: %v\n", trxf_connection_uuid, attemptNum, err)
				retryCount++
				attemptNum++
				state.mu.Lock()
				state.RetryCount = retryCount
				state.AttemptNumber = attemptNum
				state.LastAttemptTime = time.Now()
				state.mu.Unlock()
				continue
			case <-time.After(ConnectionTimeout):
				fmt.Printf("P2P transfer %s timed out on attempt %d\n", trxf_connection_uuid, attemptNum)
				retryCount++
				attemptNum++
				state.mu.Lock()
				state.RetryCount = retryCount
				state.AttemptNumber = attemptNum
				state.LastAttemptTime = time.Now()
				state.mu.Unlock()
				continue
			case <-ctx.Done():
				return
			}
		case success := <-state.successCh:
			if success {
				state.mu.Lock()
				state.Status = "success"
				state.mu.Unlock()
				fmt.Printf("P2P transfer %s succeeded on attempt %d\n", trxf_connection_uuid, attemptNum)
				return
			}
		case err := <-state.failureCh:
			fmt.Printf("P2P transfer %s failed on attempt %d: %v\n", trxf_connection_uuid, attemptNum, err)
			retryCount++
			attemptNum++
			state.mu.Lock()
			state.RetryCount = retryCount
			state.AttemptNumber = attemptNum
			state.LastAttemptTime = time.Now()
			state.mu.Unlock()
			select {
			case <-time.After(backoffDelay):
				continue
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
	fmt.Printf("P2P transfer %s failed after %d attempts, switching to relay\n", trxf_connection_uuid, MaxRetries)
	p.switchToRelay(trxf_connection_uuid)
}

// sendP2PInitiation sends P2P initiation message to both agents
func (p *P2PManager) sendP2PInitiation(trxf_connection_uuid, requestingAgent, sourceAgent, requestingEndpoint, sourceEndpoint string, attemptNumber int) error {
	requestingMsg := models.Message{
		Type: models.MasterMsgTransferStatus,
		Payload: map[string]interface{}{
			"transfer_mode":      "p2p",
			"trxf_connection_id": trxf_connection_uuid,
			"target_agent_id":    sourceAgent,
			"target_endpoint":    sourceEndpoint,
			"attempt_number":     attemptNumber,
			"max_attempts":       MaxRetries,
			"countdown_seconds":  3,
		},
	}
	p.wshub.Send(requestingAgent, Outbound{Msg: &requestingMsg})
	sourceMsg := models.Message{
		Type: models.MasterMsgTransferStatus,
		Payload: map[string]interface{}{
			"transfer_mode":      "p2p",
			"trxf_connection_id": trxf_connection_uuid,
			"target_agent_id":    requestingAgent,
			"target_endpoint":    requestingEndpoint,
			"attempt_number":     attemptNumber,
			"max_attempts":       MaxRetries,
			"countdown_seconds":  3,
		},
	}
	p.wshub.Send(sourceAgent, Outbound{Msg: &sourceMsg})
	fmt.Printf("Triggered P2P initiation for %s (attempt %d) to %s and %s\n", trxf_connection_uuid, attemptNumber, requestingAgent, sourceAgent)
	return nil
}

// HandleP2PSuccess is called when an agent reports P2P connection success
func (p *P2PManager) HandleP2PSuccess(connectionID string) {
	p.mu.RLock()
	state := p.activeTransfers[connectionID]
	p.mu.RUnlock()
	if state == nil {
		fmt.Printf("Received P2P success for unknown connection: %s\n", connectionID)
		return
	}
	state.mu.Lock()
	if state.Status == "success" {
		state.mu.Unlock()
		return
	}
	state.Status = "success"
	state.mu.Unlock()
	select {
	case state.successCh <- true:
	default:
	}
	fmt.Printf("P2P transfer %s marked as successful\n", connectionID)
}

// HandleP2PFailure is called when an agent reports P2P connection failure
func (p *P2PManager) HandleP2PFailure(connectionID string, reason string) {
	p.mu.RLock()
	state := p.activeTransfers[connectionID]
	p.mu.RUnlock()
	if state == nil {
		fmt.Printf("Received P2P failure for unknown connection: %s\n", connectionID)
		return
	}
	select {
	case state.failureCh <- errors.New(reason):
	default:
	}
	fmt.Printf("P2P transfer %s reported failure: %s\n", connectionID, reason)
}

// switchToRelay switches the transfer to relay mode
func (p *P2PManager) switchToRelay(trxf_connection_uuid string) {
	p.mu.RLock()
	state := p.activeTransfers[trxf_connection_uuid]
	p.mu.RUnlock()
	if state == nil {
		return
	}
	state.mu.Lock()
	if state.Status == "relay" {
		state.mu.Unlock()
		return // Already switched to relay
	}
	state.Status = "relay"
	state.mu.Unlock()
	p.wshub.Mutex.RLock()
	sourceConn := p.wshub.Connections[state.SourceAgent]
	p.wshub.Mutex.RUnlock()
	if sourceConn != nil {
		sourceConn.ConnMutex.Lock()
		if sourceConn.RelayTo == "" {
			sourceConn.RelayTo = state.RequestingAgent
		}
		sourceConn.ConnMutex.Unlock()
		relayMsg := models.Message{
			Type: models.MasterMsgTransferStatus,
			Payload: map[string]interface{}{
				"trxf_connection_id":  trxf_connection_uuid,
				"requesting_agent_id": state.RequestingAgent,
				"fallback":            true,
				"transfer_mode":       "relay",
			},
		}
		p.wshub.Send(state.SourceAgent, Outbound{Msg: &relayMsg})
		requestingMsg := models.Message{
			Type: models.MasterMsgTransferStatus,
			Payload: map[string]interface{}{
				"trxf_connection_id": trxf_connection_uuid,
				"fallback":           true,
				"transfer_mode":      "relay",
			},
		}
		p.wshub.Send(state.RequestingAgent, Outbound{Msg: &requestingMsg})
		fmt.Printf("Switched transfer %s to relay mode (%s -> %s)\n", trxf_connection_uuid, state.SourceAgent, state.RequestingAgent)
	}
	state.CancelFunc()
}

// getAgentEndpoint retrieves the public endpoint for an agent
func (p *P2PManager) getAgentEndpoint(agentID string) (string, error) {
	p.wshub.Mutex.RLock()
	conn := p.wshub.Connections[agentID]
	p.wshub.Mutex.RUnlock()
	if conn == nil {
		return "", fmt.Errorf("agent %s not connected", agentID)
	}
	conn.ConnMutex.RLock()
	endpoint := conn.PublicEndpoint
	conn.ConnMutex.RUnlock()
	if endpoint == "" {
		return "", fmt.Errorf("agent %s has no public endpoint", agentID)
	}
	return endpoint, nil
}
