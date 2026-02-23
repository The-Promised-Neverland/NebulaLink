package transfer

import (
	"github.com/The-Promised-Neverland/master-server/internal/models"
)

type TransferMode string
const (
	ModeP2P   TransferMode = "p2p"
	ModeRelay TransferMode = "relay"
)

type Outbound struct {
	Msg    *models.Message
	Binary []byte
}


type MessageSender interface {
	Send(agentID string, msg Outbound)
}

type ConnectionGetter interface {
	GetConnection(agentID string) ConnectionInfo
}

type ConnectionInfo interface {
	GetPublicEndpoint() string
	SetRelayTo(agentID string)
}
