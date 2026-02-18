package agentworker

import (
	"time"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/internal/ws"
)

type ServiceProvider interface {
	GetHostMetrics() *models.HostMetrics
}

type AgentWorker struct {
	Agent   *ws.Agent
	Service ServiceProvider
	Cfg     *config.Config
}

func NewAgentWorker(agent *ws.Agent, provider *service.Service, Config *config.Config) *AgentWorker {
	return &AgentWorker{
		Agent:   agent,
		Service: provider,
		Cfg:     Config,
	}
}

func (w *AgentWorker) SendHeartbeat() error {
	metrics := w.Service.GetHostMetrics()
	msg := models.Message{
		Type: models.AgentMsgHeartbeat,
		Payload: models.Metrics{
			AgentID:    w.Cfg.AgentID(),
			AgentName:  w.Cfg.AgentName(),
			SysMetrics: *metrics,
			Timestamp:  time.Now().Unix(),
		},
	}
	return w.Agent.Send(msg)
}

func (w *AgentWorker) SendConnSeverNotice() error {
	msg := models.Message{
		Type: models.AgentConnBreakNotice,
		Payload: models.ConnBreak{
			AgentID:   w.Cfg.AgentID(),
			Timestamp: time.Now().Unix(),
		},
	}
	return w.Agent.Send(msg)
}

func (w *AgentWorker) SendDirectorySnapshot(snapshot models.DirectorySnapshot) error {
	msg := models.Message{
		Type:    models.AgentMsgDirectorySnapshot,
		Payload: snapshot,
	}
	return w.Agent.Send(msg)
}
