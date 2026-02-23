package transfer

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
	ConnectionTimeout = 30 * time.Second
	InitialBackoff    = 1 * time.Second
)

type FailedTransferInfo struct {
	ConnectionID    string
	RequestingAgent string
	SourceAgent     string
}

type P2PTransferState struct {
	ConnectionID        string
	RequestingAgent     string
	SourceAgent         string
	Path                string // File path to transfer
	AttemptNumber       int
	RetryCount          int
	MaxRetries          int
	Status              string // "attempting", "connected", "ready", "relay", "failed"
	StartTime           time.Time
	LastAttemptTime     time.Time
	CancelFunc          context.CancelFunc
	successCh           chan bool
	failureCh           chan error
	requestingConfirmed bool
	sourceConfirmed     bool
	mu                  sync.RWMutex
}

type P2PConnectionConfirmed struct {
	ConnectionID    string
	RequestingAgent string
	SourceAgent     string
	Path            string
}

type P2PConnectionFailed struct {
	ConnectionID    string
	RequestingAgent string
	SourceAgent     string
	Path            string
	Reason          string
}

type P2PCoordinator struct {
	activeTransfers     map[string]*P2PTransferState
	messageSender       MessageSender
	connGetter          ConnectionGetter
	connectionConfirmed chan P2PConnectionConfirmed // Channel to notify when both agents confirm
	connectionFailed    chan P2PConnectionFailed    // Channel to notify when P2P connection fails
	mu                  sync.RWMutex
}

func NewP2PCoordinator(messageSender MessageSender, connGetter ConnectionGetter, confirmedCh chan P2PConnectionConfirmed, failedCh chan P2PConnectionFailed) *P2PCoordinator {
	return &P2PCoordinator{
		activeTransfers:     make(map[string]*P2PTransferState),
		messageSender:       messageSender,
		connGetter:          connGetter,
		connectionConfirmed: confirmedCh,
		connectionFailed:    failedCh,
	}
}

func (p *P2PCoordinator) GetMode() TransferMode {
	return ModeP2P
}

func (p *P2PCoordinator) AttemptP2PConnection(requestingAgentID, sourceAgentID, path string) (string, bool) {
	_, err1 := p.GetAgentEndpoint(requestingAgentID)
	_, err2 := p.GetAgentEndpoint(sourceAgentID)
	if err1 != nil || err2 != nil {
		fmt.Printf("P2P connection attempt failed: endpoints not available: requesting=%v, source=%v\n", err1, err2)
		return "", false
	}
	connectionID, err := p.StartP2PConnectionTest(requestingAgentID, sourceAgentID, path)
	if err != nil {
		fmt.Printf("P2P connection attempt failed to start: %v\n", err)
		return "", false
	}
	fmt.Printf("P2P connection attempt started: connection_id=%s, %s -> %s\n", connectionID, sourceAgentID, requestingAgentID)
	return connectionID, true
}

func (p *P2PCoordinator) StartP2PConnectionTest(requestingAgent, sourceAgent, path string) (string, error) {
	connectionID := uuid.New().String()
	requestingEndpoint, err := p.GetAgentEndpoint(requestingAgent)
	if err != nil {
		return "", fmt.Errorf("failed to get requesting agent endpoint: %w", err)
	}
	sourceEndpoint, err := p.GetAgentEndpoint(sourceAgent)
	if err != nil {
		return "", fmt.Errorf("failed to get source agent endpoint: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	state := &P2PTransferState{
		ConnectionID:        connectionID,
		RequestingAgent:     requestingAgent,
		SourceAgent:         sourceAgent,
		Path:                path,
		AttemptNumber:       1,
		RetryCount:          0,
		MaxRetries:          MaxRetries,
		Status:              "attempting",
		StartTime:           time.Now(),
		LastAttemptTime:     time.Now(),
		CancelFunc:          cancel,
		successCh:           make(chan bool, 1),
		failureCh:           make(chan error, 1),
		requestingConfirmed: false,
		sourceConfirmed:     false,
	}
	p.mu.Lock()
	p.activeTransfers[connectionID] = state
	p.mu.Unlock()
	go p.testConnectionWithRetries(ctx, connectionID, requestingEndpoint, sourceEndpoint)
	return connectionID, nil
}

func (p *P2PCoordinator) testConnectionWithRetries(ctx context.Context, connectionID, requestingEndpoint, sourceEndpoint string) {
	p.mu.RLock()
	state := p.activeTransfers[connectionID]
	p.mu.RUnlock()
	if state == nil {
		return
	}
	attemptNum := 1
	retryCount := 0
	for retryCount < MaxRetries {
		err := p.sendP2PInitiation(connectionID, state.RequestingAgent, state.SourceAgent, requestingEndpoint, sourceEndpoint, attemptNum)
		if err != nil {
			fmt.Printf("Failed to send P2P initiation for %s: %v\n", connectionID, err)
			retryCount++
			attemptNum++
			continue
		}
		select {
		case success := <-state.successCh:
			if success {
				state.mu.Lock()
				state.Status = "connected"
				state.mu.Unlock()
				fmt.Printf("P2P connection %s confirmed on attempt %d - keeping state until transfer completes\n", connectionID, attemptNum)
				<-ctx.Done()
				return
			}
		case err := <-state.failureCh:
			fmt.Printf("P2P connection %s failed on attempt %d: %v\n", connectionID, attemptNum, err)
			retryCount++
			attemptNum++
			state.mu.Lock()
			state.RetryCount = retryCount
			state.AttemptNumber = attemptNum
			state.LastAttemptTime = time.Now()
			state.mu.Unlock()
			backoffDelay := time.Duration(math.Pow(2, float64(retryCount))) * InitialBackoff
			select {
			case <-time.After(backoffDelay):
				continue
			case <-ctx.Done():
				return
			}
		case <-time.After(ConnectionTimeout):
			fmt.Printf("P2P connection %s timed out on attempt %d\n", connectionID, attemptNum)
			retryCount++
			attemptNum++
			state.mu.Lock()
			state.RetryCount = retryCount
			state.AttemptNumber = attemptNum
			state.LastAttemptTime = time.Now()
			state.mu.Unlock()
			backoffDelay := time.Duration(math.Pow(2, float64(retryCount))) * InitialBackoff
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
	fmt.Printf("P2P transfer %s failed after %d attempts\n", connectionID, MaxRetries)
	p.mu.RLock()
	state = p.activeTransfers[connectionID]
	p.mu.RUnlock()
	if state != nil {
		state.CancelFunc()
		state.mu.Lock()
		state.Status = "failed"
		path := state.Path
		requestingAgent := state.RequestingAgent
		sourceAgent := state.SourceAgent
		state.mu.Unlock()
		p.mu.Lock()
		delete(p.activeTransfers, connectionID)
		p.mu.Unlock()
		// Notify TransferManager that P2P connection failed
		if p.connectionFailed != nil {
			select {
			case p.connectionFailed <- P2PConnectionFailed{
				ConnectionID:    connectionID,
				RequestingAgent: requestingAgent,
				SourceAgent:     sourceAgent,
				Path:            path,
				Reason:          "max retries exceeded",
			}:
			default:
				fmt.Printf("Warning: connection failed channel full for %s\n", connectionID)
			}
		}
	}
}

func (p *P2PCoordinator) sendP2PInitiation(connectionID, requestingAgent, sourceAgent, requestingEndpoint, sourceEndpoint string, attemptNumber int) error {
	fmt.Printf("Sending P2P initiation to both agents (attempt %d)\n", attemptNumber)
	requestingMsg := models.Message{
		Type: models.MasterMsgP2PInitiate,
		Payload: map[string]interface{}{
			"connection_id":     connectionID,
			"target_agent_id":   sourceAgent,
			"target_endpoint":   sourceEndpoint,
			"attempt_number":    attemptNumber,
			"max_attempts":      3,
			"countdown_seconds": 3,
		},
	}
	p.messageSender.Send(requestingAgent, Outbound{Msg: &requestingMsg})
	sourceMsg := models.Message{
		Type: models.MasterMsgP2PInitiate,
		Payload: map[string]interface{}{
			"connection_id":     connectionID,
			"target_agent_id":   requestingAgent,
			"target_endpoint":   requestingEndpoint,
			"attempt_number":    attemptNumber,
			"max_attempts":      3,
			"countdown_seconds": 3,
		},
	}
	p.messageSender.Send(sourceAgent, Outbound{Msg: &sourceMsg})
	fmt.Printf("P2P initiation messages sent to both agents\n")
	return nil
}

// HandleP2PSuccess is called when an agent reports P2P connection success
func (p *P2PCoordinator) HandleP2PSuccess(connectionID string, agentID string) {
	p.mu.RLock()
	state := p.activeTransfers[connectionID]
	p.mu.RUnlock()
	if state == nil {
		fmt.Printf("Received P2P success for unknown connection: %s from agent %s\n", connectionID, agentID)
		return
	}
	state.mu.Lock()
	switch agentID {
	case state.RequestingAgent:
		state.requestingConfirmed = true
		fmt.Printf("Requesting agent %s confirmed P2P connection for %s\n", agentID, connectionID)
	case state.SourceAgent:
		state.sourceConfirmed = true
		fmt.Printf("Source agent %s confirmed P2P connection for %s\n", agentID, connectionID)
	default:
		state.mu.Unlock()
		fmt.Printf("Unknown agent %s confirmed P2P for connection %s\n", agentID, connectionID)
		return
	}
	bothConfirmed := state.requestingConfirmed && state.sourceConfirmed
	if bothConfirmed {
		state.Status = "connected"
		path := state.Path
		sourceAgent := state.SourceAgent
		requestingAgent := state.RequestingAgent
		state.mu.Unlock()
		select {
		case state.successCh <- true:
		default:
		}
		fmt.Printf("Both agents confirmed P2P connection for %s - notifying TransferManager\n", connectionID)
		if p.connectionConfirmed != nil {
			select {
			case p.connectionConfirmed <- P2PConnectionConfirmed{
				ConnectionID:    connectionID,
				RequestingAgent: requestingAgent,
				SourceAgent:     sourceAgent,
				Path:            path,
			}:
			default:
				fmt.Printf("Warning: connection confirmed channel full for %s\n", connectionID)
			}
		}
	} else {
		state.mu.Unlock()
		fmt.Printf("Waiting for other agent to confirm P2P connection for %s (requesting: %v, source: %v)\n",
			connectionID, state.requestingConfirmed, state.sourceConfirmed)
	}
}

// HandleP2PFailure is called when an agent reports P2P connection failure
func (p *P2PCoordinator) HandleP2PFailure(connectionID string, reason string) {
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
		// TODO: Channel full
	}
	fmt.Printf("P2P transfer %s reported failure: %s\n", connectionID, reason)
}

// GetAgentEndpoint retrieves the public endpoint for an agent
func (p *P2PCoordinator) GetAgentEndpoint(agentID string) (string, error) {
	if p.connGetter == nil {
		return "", fmt.Errorf("connection getter not initialized")
	}
	connInfo := p.connGetter.GetConnection(agentID)
	if connInfo == nil {
		return "", fmt.Errorf("agent %s not connected", agentID)
	}
	endpoint := connInfo.GetPublicEndpoint()
	if endpoint == "" {
		return "", fmt.Errorf("agent %s has no public endpoint", agentID)
	}
	return endpoint, nil
}

// GetTransferState returns the current state of a transfer
func (p *P2PCoordinator) GetTransferState(connectionID string) *P2PTransferState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.activeTransfers[connectionID]
}

// GetFailedTransfer returns transfer info if P2P failed, nil otherwise
func (p *P2PCoordinator) GetFailedTransfer(connectionID string) *FailedTransferInfo {
	p.mu.RLock()
	state := p.activeTransfers[connectionID]
	p.mu.RUnlock()
	if state == nil {
		return nil
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	if state.Status != "failed" {
		return nil
	}
	return &FailedTransferInfo{
		ConnectionID:    state.ConnectionID,
		RequestingAgent: state.RequestingAgent,
		SourceAgent:     state.SourceAgent,
	}
}

func (p *P2PCoordinator) RemoveTransfer(connectionID string) {
	p.mu.Lock()
	delete(p.activeTransfers, connectionID)
	p.mu.Unlock()
}
