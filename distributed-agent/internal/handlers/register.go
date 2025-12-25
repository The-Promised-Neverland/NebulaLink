package handlers

import (
	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/control"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/internal/ws"
)

type Handlers struct {
	Agent      *ws.Agent
	Service    *service.Service
	Config     *config.Config
	Controller control.AgentController
}

func NewHandler(
	agent *ws.Agent,
	svc *service.Service,
	cfg *config.Config,
	ctrl control.AgentController,
) *Handlers {
	return &Handlers{
		Agent:      agent,
		Service:    svc,
		Config:     cfg,
		Controller: ctrl,
	}
}

func (h *Handlers) RegisterHandlers() {
	h.Agent.RegisterHandler(models.MasterMsgMetricsRequest, func(msg *any) error {
		return h.RequestMetrics()
	})

	h.Agent.RegisterHandler(models.MasterMsgTaskAssignment, func(msg *any) error {
		return h.AssignTask(msg)
	})

	h.Agent.RegisterHandler(models.MasterMsgRestartAgent, func(msg *any) error {
		return h.RestartAgent()
	})

	h.Agent.RegisterHandler(models.MasterMsgAgentUninstall, func(msg *any) error {
		return h.UninstallAgent()
	})
}
