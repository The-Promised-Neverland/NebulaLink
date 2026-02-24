package handlers

import (
	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/internal/transfer"
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
	TransferManager      *transfer.TransferManager
}

func NewHandler(agent *ws.Agent, businessService *service.Service, cfg *config.Config, daemonManagerService DaemonManagerService) *Handlers {
	transferManager := transfer.NewTransferManager(cfg, businessService, agent)
	return &Handlers{
		Agent:                agent,
		BusinessService:      businessService,
		Config:               cfg,
		DaemonManagerService: daemonManagerService,
		TransferManager:      transferManager,
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

	h.Agent.RegisterHandler(models.MasterMsgTransferIntent, func(msg *any) error {
		return h.LogTransferIntent(msg)
	})

	h.Agent.RegisterHandler(models.MasterMsgP2PTransferStart, func(msg *any) error {
		return h.SendFileSystem(msg)
	})

	h.Agent.RegisterHandler(models.MasterMsgRelayTransferStart, func(msg *any) error {
		return h.SendFileSystem(msg)
	})

	h.Agent.RegisterHandler(models.MasterMsgTransferStatus, func(msg *any) error {
		return h.ReceiveTransfer(msg)
	})

	h.Agent.RegisterHandler(models.MasterMsgP2PInitiate, func(msg *any) error {
		return h.HandleP2PInitiation(msg)
	})

	h.Agent.RegisterHandler(models.MasterMsgRelayFallback, func(msg *any) error {
		return h.HandleRelayFallback(msg)
	})
}
