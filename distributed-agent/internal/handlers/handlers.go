package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/gorilla/websocket"
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
	return h.DaemonManagerService.RestartDaemon()
}

func (h *Handlers) UninstallAgent() error {
	return h.DaemonManagerService.UninstallDaemon()
}

func (h *Handlers) SendFileSystem(msg *interface{}) error {
	m, ok := (*msg).(map[string]interface{})
	if !ok {
		return errors.New("invalid message format")
	}
	payloadRaw, ok := m["payload"]
	if !ok {
		return errors.New("missing payload")
	}
	path, ok := payloadRaw.(string)
	if !ok {
		return fmt.Errorf("payload is not a string")
	}
	path = filepath.Clean(path)
	dataCh, errCh := h.BusinessService.StreamRequestedFileSystem(path)
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan struct{})
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				statusMsg := models.Message{
					Type: models.MasterMsgFileSystemRequest,
					Payload: map[string]string{
						"status": "running",
					},
				}
				statusBytes, _ := json.Marshal(statusMsg)
				h.Agent.Conn.WriteMessage(websocket.BinaryMessage, statusBytes)
			}
		}
	}()
	writer, err := h.Agent.Conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		close(done)
		return err
	}
	for chunk := range dataCh {
		_, err := writer.Write(chunk)
		if err != nil {
			close(done)
			return err
		}
	}
	writer.Close()
	close(done)
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	default:
	}
	doneMsg := models.Message{
		Type: models.MasterMsgFileSystemRequest,
		Payload: map[string]string{
			"status": "completed",
		},
	}
	statusBytes, _ := json.Marshal(doneMsg)
	h.Agent.Conn.WriteMessage(websocket.BinaryMessage, statusBytes)
	return nil
}
