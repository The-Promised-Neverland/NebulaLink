package transfer

import (
	"fmt"

	"github.com/The-Promised-Neverland/master-server/internal/models"
)

type TransferManager struct {
	p2pCoordinator      *P2PCoordinator
	relayCoordinator    *RelayCoordinator
	messageSender       MessageSender
	connGetter          ConnectionGetter
	p2pConfirmedChannel chan P2PConnectionConfirmed
	p2pFailedChannel    chan P2PConnectionFailed
}

func NewTransferManager(messageSender MessageSender, connGetter ConnectionGetter) *TransferManager {
	p2pConfirmedCh := make(chan P2PConnectionConfirmed, 5)
	p2pFailedCh := make(chan P2PConnectionFailed, 5)
	manager := &TransferManager{
		messageSender:       messageSender,
		connGetter:          connGetter,
		p2pConfirmedChannel: p2pConfirmedCh,
		p2pFailedChannel:    p2pFailedCh,
		p2pCoordinator:      NewP2PCoordinator(messageSender, connGetter, p2pConfirmedCh, p2pFailedCh),
		relayCoordinator:    NewRelayCoordinator(messageSender, connGetter),
	}
	go manager.handleP2PConfirmations()
	go manager.handleP2PFailures()
	return manager
}

func (m *TransferManager) handleP2PConfirmations() {
	for confirmed := range m.p2pConfirmedChannel {
		fmt.Printf("[TRANSFER] COMPLETE: P2P connection confirmed by both agents, connection_id=%s, source_agent=%s -> requesting_agent=%s\n", confirmed.ConnectionID, confirmed.SourceAgent, confirmed.RequestingAgent)
		fmt.Printf("[TRANSFER] Master giving green signal to transfer - sending file transfer request to source_agent=%s\n", confirmed.SourceAgent)
		m.InitiateP2PTransfer(confirmed)
	}
}

func (m *TransferManager) handleP2PFailures() {
	for failed := range m.p2pFailedChannel {
		fmt.Printf("[TRANSFER] FAILED: P2P connection failed, connection_id=%s, source_agent=%s -> requesting_agent=%s, reason=%s\n",
			failed.ConnectionID, failed.SourceAgent, failed.RequestingAgent, failed.Reason)
		fmt.Printf("[TRANSFER] Falling back to relay mode, connection_id=%s\n", failed.ConnectionID)
		payloadMap := map[string]interface{}{
			"requesting_agent_id": failed.RequestingAgent,
			"connection_id":       failed.ConnectionID,
		}
		if failed.Path != "" {
			payloadMap["path"] = failed.Path
		}
		if _, err := m.relayCoordinator.InitiateTransfer(failed.RequestingAgent, failed.SourceAgent, payloadMap); err != nil {
			fmt.Printf("[TRANSFER] FAILED: Relay fallback initiation failed: %v\n", err)
		} else {
			fmt.Printf("[TRANSFER] SUCCESS: Relay fallback initiated successfully\n")
		}
	}
}

// NotifyTransferIntent sends transfer intent notification to both agents
func (m *TransferManager) NotifyTransferIntent(requestingAgentID, sourceAgentID, path, connectionID string) {
	intentPayload := map[string]interface{}{
		"requesting_agent_id": requestingAgentID,
		"source_agent_id":     sourceAgentID,
		"path":                path,
	}
	if connectionID != "" {
		intentPayload["connection_id"] = connectionID
	}
	intentMsg := models.Message{
		Type:    models.MasterMsgTransferIntent,
		Payload: intentPayload,
	}
	m.messageSender.Send(requestingAgentID, Outbound{Msg: &intentMsg})
	m.messageSender.Send(sourceAgentID, Outbound{Msg: &intentMsg})
	fmt.Printf("[AUDIT] Transfer intent sent to requesting_agent=%s\n", requestingAgentID)
	fmt.Printf("[AUDIT] Transfer intent sent to source_agent=%s\n", sourceAgentID)
}

func (m *TransferManager) HandleAgentRequestFile(msg *models.Message, sourceAgentID string) error {
	payloadMap, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload format")
	}
	requestingAgentID, ok := payloadMap["requesting_agent_id"].(string)
	if !ok || requestingAgentID == "" {
		return fmt.Errorf("requesting_agent_id is missing")
	}
	path, _ := payloadMap["path"].(string)
	fmt.Printf("[TRANSFER] File transfer request received: requesting_agent=%s wants file from source_agent=%s, path=%s\n", requestingAgentID, sourceAgentID, path)
	connectionID := ""
	if id, ok := payloadMap["connection_id"].(string); ok && id != "" {
		connectionID = id
	}
	m.NotifyTransferIntent(requestingAgentID, sourceAgentID, path, connectionID)
	fmt.Printf("[TRANSFER] Attempting P2P connection between requesting_agent=%s and source_agent=%s\n", requestingAgentID, sourceAgentID)
	connectionID, connectionOK := m.p2pCoordinator.AttemptP2PConnection(requestingAgentID, sourceAgentID, path)
	if !connectionOK {
		fmt.Printf("[TRANSFER] FAILED: P2P connection attempt failed (endpoints not available), falling back to relay mode\n")
		fmt.Printf("[TRANSFER] Initiating relay transfer: source_agent=%s -> requesting_agent=%s\n", sourceAgentID, requestingAgentID)
		_, relayErr := m.relayCoordinator.InitiateTransfer(requestingAgentID, sourceAgentID, payloadMap)
		if relayErr != nil {
			fmt.Printf("[TRANSFER] FAILED: Relay transfer initiation failed: %v\n", relayErr)
		} else {
			fmt.Printf("[TRANSFER] SUCCESS: Relay transfer initiated successfully\n")
		}
		return relayErr
	}
	fmt.Printf("[TRANSFER] SUCCESS: P2P connection attempt started, connection_id=%s, waiting for both agents to confirm...\n", connectionID)
	return nil
}

func (m *TransferManager) GetP2PCoordinator() *P2PCoordinator {
	return m.p2pCoordinator
}

func (m *TransferManager) GetRelayCoordinator() *RelayCoordinator {
	return m.relayCoordinator
}

func (m *TransferManager) HandleP2PFailureFallback(connectionID string) {
	failedTransfer := m.p2pCoordinator.GetFailedTransfer(connectionID)
	if failedTransfer == nil {
		return
	}
	fmt.Printf("P2P transfer %s failed, initiating relay fallback\n", connectionID)
	if err := m.initiateRelayFallback(failedTransfer.ConnectionID, failedTransfer.RequestingAgent, failedTransfer.SourceAgent); err != nil {
		fmt.Printf("Failed to initiate relay fallback: %v\n", err)
		return
	}
	m.p2pCoordinator.RemoveTransfer(connectionID)
}

// initiateRelayFallback handles fallback from P2P to relay
func (m *TransferManager) initiateRelayFallback(connectionID, requestingAgentID, sourceAgentID string) error {
	fmt.Printf("[TRANSFER] P2P failed, switching to relay mode, connection_id=%s, source_agent=%s -> requesting_agent=%s\n",
		connectionID, sourceAgentID, requestingAgentID)
	if m.connGetter == nil {
		return fmt.Errorf("connection getter not initialized")
	}
	sourceConn := m.connGetter.GetConnection(sourceAgentID)
	if sourceConn == nil {
		return fmt.Errorf("source agent %s not connected", sourceAgentID)
	}
	sourceConn.SetRelayTo(requestingAgentID)
	relayMsg := models.Message{
		Type: models.MasterMsgRelayFallback,
		Payload: map[string]interface{}{
			"connection_id":       connectionID,
			"requesting_agent_id": requestingAgentID,
			"source_agent_id":     sourceAgentID,
			"transfer_mode":       "relay",
			"fallback":            true,
			"action":              "send",
		},
	}
	m.messageSender.Send(sourceAgentID, Outbound{Msg: &relayMsg})
	fmt.Printf("[TRANSFER] Sent relay fallback command to SOURCE agent=%s (action=SEND), connection_id=%s\n", sourceAgentID, connectionID)
	requestingMsg := models.Message{
		Type: models.MasterMsgRelayFallback,
		Payload: map[string]interface{}{
			"connection_id":       connectionID,
			"requesting_agent_id": requestingAgentID,
			"source_agent_id":     sourceAgentID,
			"transfer_mode":       "relay",
			"fallback":            true,
			"action":              "receive",
		},
	}
	m.messageSender.Send(requestingAgentID, Outbound{Msg: &requestingMsg})
	fmt.Printf("[TRANSFER] Sent relay fallback command to REQUESTING agent=%s (action=RECEIVE), connection_id=%s\n", requestingAgentID, connectionID)
	fmt.Printf("[TRANSFER] Relay mode activated: source_agent=%s -> requesting_agent=%s\n", sourceAgentID, requestingAgentID)
	return nil
}
