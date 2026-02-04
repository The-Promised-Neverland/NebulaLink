package ws_mssg_processor

import (
	"github.com/The-Promised-Neverland/master-server/internal/models"
)

type RoutedMessage struct {
	TargetId  string // "frontend", "agentID",
	Message models.Message
}

type Processor struct {
	OutgoingCh chan RoutedMessage
}

func NewProcessor() *Processor {
	return &Processor{
		OutgoingCh: make(chan RoutedMessage),
	}
}

// Delegates message based on actor
func (p *Processor) ProcessMessage(source string, msg *models.Message) {
	if source == "frontend" {
		p.handleFrontendMessage(msg)
	} else {
		p.handleAgentMessage(msg)
	}
}

// Handles agentic tasks
func (p *Processor) handleAgentMessage(msg *models.Message) {
	switch msg.Type {
	case "agent_metrics", "task_result", "agent_directory_snapshot":
		p.OutgoingCh <- RoutedMessage{TargetId: "frontend", Message: *msg}
	default:
	}
}

// Handles frontend tasks
func (p *Processor) handleFrontendMessage(msg *models.Message) {
	// TODO
}
