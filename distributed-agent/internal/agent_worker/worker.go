package agentworker

import (
	"time"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/internal/ws"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

type ServiceProvider interface {
	GetHostMetrics() (*models.HostMetrics, error)
}

type AgentWorker struct {
	Agent   *ws.Agent
	Service *service.Service
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
	logger.Log.Info("Sending Heartbeat", metrics);
	msg := models.Message{
		Type: models.AgentMsgHeartbeat,
		Payload: models.Metrics{
			AgentID:    w.Cfg.AgentID(),
			SysMetrics: *metrics,
			Timestamp:  time.Now().Unix(),
		},
	}
	return w.Agent.Send(msg)
}

func (w *AgentWorker) SendJobUpdate(jobID string, status string) error {
	// TODO
	return nil
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
