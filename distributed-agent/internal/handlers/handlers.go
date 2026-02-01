package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

func (h *Handlers) RequestMetrics() error {
	metrics := h.BusinessService.GetHostMetrics()
	response := models.Message{
		Type:    models.MasterMsgMetricsRequest,
		Payload: metrics,
	}
	jsonData, err := json.Marshal(response)
	if err != nil {
		logger.Log.Error("failed to marshal metrics: %w", "err", err)
		return err
	}
	url := h.Config.MasterServerConn() + "/api/v1/callback/metrics/" + h.Config.AgentID()
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
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
