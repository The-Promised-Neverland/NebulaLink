package ws_mssg_processor

import (
	"fmt"
	"strings"

	"github.com/The-Promised-Neverland/master-server/internal/models"
)

type RoutedMessage struct {
	Target  string // "frontend", "agent:ID",
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

// source format: "agent:ID" or "frontend"
func (p *Processor) ProcessMessage(source string, msg *models.Message) {
	fmt.Print("Recieved msg", msg, " from ", source);
	if strings.HasPrefix(source, "agent:") {
		p.handleAgentMessage(msg)
	} else if source == "frontend" {
		p.handleFrontendMessage(msg)
	}
}

func (p *Processor) handleAgentMessage(msg *models.Message) {
	switch msg.Type {
	case "agent_metrics", "task_result":
		p.OutgoingCh <- RoutedMessage{Target: "frontend", Message: *msg}
	default:
	}
}

func (p *Processor) handleFrontendMessage(msg *models.Message) {
	// TODO
}

