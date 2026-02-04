package handlers

import "github.com/The-Promised-Neverland/agent/internal/models"

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
	return h.DaemonControl.Restart()
}

func (h *Handlers) UninstallAgent() error {
	return h.DaemonControl.Uninstall()
}
