package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/internal/ws"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

func (h *Handlers) RequestMetrics() error {
	metrics := h.BusinessService.GetHostMetrics()
	response := models.Message{
		Type:    models.MasterMsgMetricsRequest,
		Payload: metrics,
	}
	return h.Agent.Send(ws.Outbound{Msg: &response})
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

func (h *Handlers) SendFileSystem(msg *any) error {
	payloadRaw, ok := (*msg).(map[string]interface{})
	if !ok {
		return errors.New("payload is not a valid map[string]interface{}")
	}
	path, ok := payloadRaw["fileSystemPath"].(string)
	if !ok {
		return fmt.Errorf("fileSystemPath is missing or not a string")
	}
	requestAgentID, ok := payloadRaw["requestedAgentID"].(string)
	if !ok {
		return fmt.Errorf("requested Agent ID is missing or not a string")
	}
	path = filepath.Clean(path)
	logger.Log.Info("Requested filePath and agentID", slog.String("filePath", path), slog.String("agentID", requestAgentID))
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
					Payload: models.FileSystemTransfer{
						AgentID:         h.Config.AgentID(),
						Status:          "running",
						AgentName:       h.Config.AgentName(),
						RequestingAgent: requestAgentID,
						Timestamp:       time.Now().Unix(),
					},
				}
				h.Agent.Send(ws.Outbound{Msg: &statusMsg})
			}
		}
	}()
	// writer, err := h.Agent.Conn.NextWriter(websocket.BinaryMessage)
	// if err != nil {
	// 	close(done)
	// 	return err
	// }
	// for chunk := range dataCh {
	// 	logger.Log.Info("Sending Bytes...", slog.Int("bytes", len(chunk)))
	// 	_, err := writer.Write(chunk)
	// 	if err != nil {
	// 		close(done)
	// 		return err
	// 	}
	// }
	// writer.Close()
	// close(done)
	// WARN: Cannot write to a websocket concurrently. Funneling is done
	starterMsg := models.Message{
		Type: models.MasterMsgFileSystemRequest,
		Payload: models.FileSystemTransfer{
			AgentID:         h.Config.AgentID(),
			Status:          "initiated",
			AgentName:       h.Config.AgentName(),
			RequestingAgent: requestAgentID,
			Timestamp:       time.Now().Unix(),
		},
	}
	h.Agent.Send(ws.Outbound{Msg: &starterMsg})
	for chunk := range dataCh {
		logger.Log.Info("Sending Bytes...", slog.Int("bytes", len(chunk)))
		h.Agent.Send(ws.Outbound{Binary: chunk})
	}
	close(done)
	select {
	case err := <-errCh:
		if err != nil {
			logger.Log.Info("Stream error", slog.String("error", err.Error()))
			return err
		}
	default:
	}
	doneMsg := models.Message{
		Type: models.MasterMsgFileSystemRequest,
		Payload: models.FileSystemTransfer{
			AgentID:         h.Config.AgentID(),
			Status:          "completed",
			AgentName:       h.Config.AgentName(),
			RequestingAgent: requestAgentID,
			Timestamp:       time.Now().Unix(),
		},
	}
	h.Agent.Send(ws.Outbound{Msg: &doneMsg})
	return nil
}
