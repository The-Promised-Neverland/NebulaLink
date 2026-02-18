package handlers

import (
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

func (h *Handlers) RequestMetrics() error {
	metrics := h.BusinessService.GetHostMetrics()
	response := models.Message{
		Type:    models.MasterMsgMetricsRequest,
		Payload: metrics,
	}
	h.Agent.Send(response)
	return nil
}

func (h *Handlers) AssignTask(msg *any) error {
	return nil
}

func (h *Handlers) RestartAgent() error {
	logger.Log.Info("Master Triggered Restart")
	return h.DaemonManagerService.Restart()
}

func (h *Handlers) UninstallAgent() error {
	return h.DaemonManagerService.Uninstall()
}
