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
		fmt.Printf("P2P connection confirmed for %s: %s -> %s, initiating file transfer\n",
			confirmed.ConnectionID, confirmed.SourceAgent, confirmed.RequestingAgent)
		m.initiateP2PTransfer(confirmed)
	}
}

func (m *TransferManager) handleP2PFailures() {
	for failed := range m.p2pFailedChannel {
		fmt.Printf("P2P connection failed for %s: %s -> %s, reason: %s, falling back to relay\n",
			failed.ConnectionID, failed.SourceAgent, failed.RequestingAgent, failed.Reason)
		// Reconstruct payload for relay fallback
		payloadMap := map[string]interface{}{
			"requesting_agent_id": failed.RequestingAgent,
			"connection_id":       failed.ConnectionID,
		}
		if failed.Path != "" {
			payloadMap["path"] = failed.Path
		}
		if _, err := m.relayCoordinator.InitiateTransfer(failed.RequestingAgent, failed.SourceAgent, payloadMap); err != nil {
			fmt.Printf("Failed to initiate relay fallback: %v\n", err)
		}
	}
}

// initiateP2PTransfer sends the file transfer request to source agent after P2P is confirmed
func (m *TransferManager) initiateP2PTransfer(confirmed P2PConnectionConfirmed) {
	fmt.Printf("Sending file transfer request to source agent %s (P2P confirmed, status=connected)\n", confirmed.SourceAgent)
	transferPayload := map[string]interface{}{
		"requesting_agent_id": confirmed.RequestingAgent,
		"connection_id":       confirmed.ConnectionID,
		"transfer_mode":       "p2p",
	}
	if confirmed.Path != "" {
		transferPayload["path"] = confirmed.Path
	}
	transferMsg := models.Message{
		Type:    models.MasterMsgAgentRequestFile,
		Payload: transferPayload,
	}
	m.messageSender.Send(confirmed.SourceAgent, Outbound{Msg: &transferMsg})
	fmt.Printf("File transfer request sent to source agent %s for connection %s (P2P confirmed)\n", confirmed.SourceAgent, confirmed.ConnectionID)
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
	fmt.Printf("File transfer request received: %s wants file from %s\n", requestingAgentID, sourceAgentID)
	connectionID, connectionOK := m.p2pCoordinator.AttemptP2PConnection(requestingAgentID, sourceAgentID, path)
	if !connectionOK {
		fmt.Printf("P2P connection attempt failed (endpoints not available), falling back to relay\n")
		_, relayErr := m.relayCoordinator.InitiateTransfer(requestingAgentID, sourceAgentID, payloadMap)
		return relayErr
	}
	fmt.Printf("P2P connection attempt started for %s, waiting for result (success/failure via channels)\n", connectionID)
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
		return // Not a failed transfer or already handled
	}
	fmt.Printf("P2P transfer %s failed, initiating relay fallback\n", connectionID)
	if err := m.initiateRelayFallback(failedTransfer.ConnectionID, failedTransfer.RequestingAgent, failedTransfer.SourceAgent); err != nil {
		fmt.Printf("Failed to initiate relay fallback: %v\n", err)
		return
	}
	// Remove the failed transfer from P2P coordinator
	m.p2pCoordinator.RemoveTransfer(connectionID)
}

// initiateRelayFallback handles fallback from P2P to relay
func (m *TransferManager) initiateRelayFallback(connectionID, requestingAgentID, sourceAgentID string) error {
	fmt.Printf("P2P failed, switching to relay mode for connection %s\n", connectionID)
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
	fmt.Printf("Sent relay fallback to SOURCE agent %s (command: SEND)\n", sourceAgentID)
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
	fmt.Printf("Sent relay fallback to DESTINATION agent %s (command: RECEIVE)\n", requestingAgentID)
	fmt.Printf("Relay mode activated for %s -> %s\n", sourceAgentID, requestingAgentID)
	return nil
}
