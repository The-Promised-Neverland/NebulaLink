package transfer

import (
	"fmt"

	"github.com/The-Promised-Neverland/master-server/internal/models"
)

type RelayCoordinator struct {
	messageSender MessageSender
	connGetter    ConnectionGetter
}

func NewRelayCoordinator(messageSender MessageSender, connGetter ConnectionGetter) *RelayCoordinator {
	return &RelayCoordinator{
		messageSender: messageSender,
		connGetter:    connGetter,
	}
}

func (r *RelayCoordinator) GetMode() TransferMode {
	return ModeRelay
}

func (r *RelayCoordinator) InitiateTransfer(requestingAgentID, sourceAgentID string, payload map[string]interface{}) (TransferMode, error) {
	fmt.Printf("[RELAY] Initiating relay transfer: source_agent=%s -> requesting_agent=%s\n", sourceAgentID, requestingAgentID)
	if r.connGetter == nil {
		return ModeRelay, fmt.Errorf("connection getter not initialized")
	}
	requestingConn := r.connGetter.GetConnection(requestingAgentID)
	sourceConn := r.connGetter.GetConnection(sourceAgentID)
	if requestingConn == nil {
		fmt.Printf("[RELAY] FAILED: Requesting agent=%s not connected\n", requestingAgentID)
		return ModeRelay, fmt.Errorf("requesting agent %s not connected", requestingAgentID)
	}
	if sourceConn == nil {
		fmt.Printf("[RELAY] FAILED: Source agent=%s not connected\n", sourceAgentID)
		return ModeRelay, fmt.Errorf("source agent %s not connected", sourceAgentID)
	}
	sourceConn.SetRelayTo(requestingAgentID)
	payload["transfer_mode"] = "relay"
	transferMsg := models.Message{
		Type:    models.MasterMsgRelayTransferStart,
		Payload: payload,
	}
	r.messageSender.Send(sourceAgentID, Outbound{Msg: &transferMsg})
	fmt.Printf("[RELAY] Relay transfer start command sent to source_agent=%s, requesting_agent=%s\n", sourceAgentID, requestingAgentID)
	return ModeRelay, nil
}

