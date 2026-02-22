package handlers

import (
	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/internal/ws"
)

// DaemonControl defines the interface for controlling the daemon lifecycle.
type DaemonManagerService interface {
	RestartDaemon() error
	UninstallDaemon() error
}

type Handlers struct {
	Agent                *ws.Agent
	BusinessService      *service.Service
	Config               *config.Config
	DaemonManagerService DaemonManagerService
}

func NewHandler(
	agent *ws.Agent,
	businessService *service.Service,
	cfg *config.Config,
	daemonManagerService DaemonManagerService,
) *Handlers {
	return &Handlers{
		Agent:                agent,
		BusinessService:      businessService,
		Config:               cfg,
		DaemonManagerService: daemonManagerService,
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

	h.Agent.RegisterHandler(models.MasterMsgAgentRequestFile, func(msg *any) error {
		return h.SendFileSystem(msg)
	})

	h.Agent.RegisterHandler(models.MasterMsgRelayManager, func(msg *any) error {
		return h.ReceiveTransfer(msg)
	})

	// P2P handlers
	h.Agent.RegisterHandler(models.MasterMsgP2PInitiate, func(msg *any) error {
		return h.HandleP2PInitiation(msg)
	})

	h.Agent.RegisterHandler(models.MasterMsgSwitchToRelay, func(msg *any) error {
		return h.HandleSwitchToRelay(msg)
	})

	h.Agent.RegisterHandler(models.MasterMsgRelayModeActivated, func(msg *any) error {
		return h.HandleRelayModeActivated(msg)
	})
}
