package ws_mssg_processor

import (
	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/The-Promised-Neverland/master-server/internal/sse"
)

type Processor struct {
	SSEHub *sse.SSEHub
}

func NewProcessor(sseHub *sse.SSEHub) *Processor {
	return &Processor{
		SSEHub: sseHub,
	}
}

func (p *Processor) ProcessAgentMessages(agentName string, msg *models.Message) {
	p.handleAgentMessage(msg)
}

// Handles agentic tasks
func (p *Processor) handleAgentMessage(msg *models.Message) {
	if p.SSEHub != nil {
		p.SSEHub.Broadcast(*msg)
	}
}
