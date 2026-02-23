package handlers

import (
	"errors"
	"fmt"
	"io"
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
	return h.DaemonManagerService.RestartDaemon()
}

func (h *Handlers) UninstallAgent() error {
	return h.DaemonManagerService.UninstallDaemon()
}

func (h *Handlers) ReceiveTransfer(msg *any) error {
	payloadRaw, ok := (*msg).(map[string]interface{})
	if !ok {
		return errors.New("payload is not a valid map[string]interface{}")
	}
	status, ok := payloadRaw["status"].(string)
	if !ok {
		return fmt.Errorf("status is missing or not a string")
	}
	sourceAgentID, ok := payloadRaw["source_agent_id"].(string)
	if !ok {
		sourceAgentID, _ = payloadRaw["agent_id"].(string)
	}
	switch status {
	case "initiated":
		if sourceAgentID == "" {
			return fmt.Errorf("source_agent_id is required to start transfer")
		}
		if err := h.Agent.StartTransfer(sourceAgentID); err != nil {
			return fmt.Errorf("failed to start transfer: %w", err)
		}
		go h.tryReceiveOverP2P(sourceAgentID)
		logger.Log.Info("Transfer initiated", "sourceAgent", sourceAgentID)
	case "completed":
		if err := h.Agent.CompleteTransfer(); err != nil {
			return fmt.Errorf("failed to complete transfer: %w", err)
		}
		logger.Log.Info("Transfer completed")
	default:
		logger.Log.Debug("Transfer status update", "status", status)
	}

	return nil
}

// tryReceiveOverP2P attempts to receive file over P2P connection
func (h *Handlers) tryReceiveOverP2P(sourceAgentID string) {
	time.Sleep(1 * time.Second)
	if h.Agent.P2PClient == nil || h.Agent.TempFile == nil {
		return
	}
	p2pConn := h.Agent.P2PClient.GetActiveConnectionByTarget(sourceAgentID)
	if p2pConn == nil || p2pConn.Status != "connected" {
		logger.Log.Debug("No active P2P connection for receiving", "sourceAgent", sourceAgentID)
		return
	}
	logger.Log.Info("Receiving file over P2P", "connection_id", p2pConn.ConnectionID, "sourceAgent", sourceAgentID)
	if err := h.Agent.P2PClient.ReceiveFileOverP2P(p2pConn.ConnectionID, h.Agent.TempFile); err != nil {
		logger.Log.Error("P2P receive failed", "error", err)
		return
	}
	if err := h.Agent.CompleteTransfer(); err != nil {
		logger.Log.Error("Failed to complete P2P transfer", "error", err)
	} else {
		logger.Log.Info("P2P file transfer completed successfully")
	}
}

func (h *Handlers) SendFileSystem(msg *any) error {
	payloadRaw, ok := (*msg).(map[string]interface{})
	if !ok {
		return errors.New("payload is not a valid map[string]interface{}")
	}
	path, ok := payloadRaw["path"].(string)
	if !ok {
		return fmt.Errorf("path is missing or not a string")
	}
	requestInitiator, ok := payloadRaw["requesting_agent_id"].(string)
	if !ok {
		return fmt.Errorf("requesting_agent_id is missing or not a string")
	}
	path = filepath.Clean(path)
	logger.Log.Info("Requested filePath and requestInitiator", slog.String("filePath", path), slog.String("requestInitiator", requestInitiator))
	p2pConn := h.Agent.P2PClient.GetActiveConnectionByTarget(requestInitiator)
	if p2pConn != nil && p2pConn.Status == "connected" {
		logger.Log.Info("Sending file over P2P", "connection_id", p2pConn.ConnectionID, "target", requestInitiator)
		dataCh, errCh := h.BusinessService.StreamRequestedFileSystem(path)
		reader := &channelReader{dataCh: dataCh, errCh: errCh}
		if err := h.Agent.P2PClient.SendFileOverP2P(p2pConn.ConnectionID, reader); err != nil {
			logger.Log.Warn("P2P send failed, falling back to relay", "error", err)
		} else {
			doneMsg := models.Message{
				Type: models.MasterMsgRelayManager,
				Payload: map[string]interface{}{
					"status":   "completed",
					"agent_id": h.Config.AgentID(),
				},
			}
			h.Agent.Send(ws.Outbound{Msg: &doneMsg})
			return nil
		}
	}
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
					Type: models.MasterMsgRelayManager,
					Payload: map[string]interface{}{
						"status":   "running",
						"agent_id": h.Config.AgentID(),
					},
				}
				h.Agent.Send(ws.Outbound{Msg: &statusMsg})
			}
		}
	}()
	starterMsg := models.Message{
		Type: models.MasterMsgRelayManager,
		Payload: map[string]interface{}{
			"status":   "initiated",
			"agent_id": h.Config.AgentID(),
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
		Type: models.MasterMsgRelayManager,
		Payload: map[string]interface{}{
			"status":   "completed",
			"agent_id": h.Config.AgentID(),
		},
	}
	h.Agent.Send(ws.Outbound{Msg: &doneMsg})
	return nil
}

// HandleP2PInitiation handles P2P initiation from master
func (h *Handlers) HandleP2PInitiation(msg *any) error {
	payloadRaw, ok := (*msg).(map[string]interface{})
	if !ok {
		return errors.New("payload is not a valid map[string]interface{}")
	}
	connectionID, ok := payloadRaw["connection_id"].(string)
	if !ok {
		return fmt.Errorf("connection_id is missing or not a string")
	}
	targetAgentID, ok := payloadRaw["target_agent_id"].(string)
	if !ok {
		return fmt.Errorf("target_agent_id is missing or not a string")
	}
	targetEndpoint, ok := payloadRaw["target_endpoint"].(string)
	if !ok {
		return fmt.Errorf("target_endpoint is missing or not a string")
	}
	attemptNumber := 1
	if an, ok := payloadRaw["attempt_number"].(float64); ok {
		attemptNumber = int(an)
	}
	countdownSeconds := 3
	if cs, ok := payloadRaw["countdown_seconds"].(float64); ok {
		countdownSeconds = int(cs)
	}
	logger.Log.Info("P2P initiation received",
		"connection_id", connectionID,
		"target_agent", targetAgentID,
		"target_endpoint", targetEndpoint,
		"attempt", attemptNumber)
	go func() {
		if h.Agent.P2PClient == nil {
			logger.Log.Error("P2P client not initialized")
			return
		}
		if err := h.Agent.P2PClient.AttemptConnection(
			connectionID,
			targetAgentID,
			targetEndpoint,
			attemptNumber,
			countdownSeconds,
		); err != nil {
			logger.Log.Error("P2P connection attempt failed", "error", err)
		}
	}()
	return nil
}

// HandleSwitchToRelay handles switch to relay mode
func (h *Handlers) HandleSwitchToRelay(msg *any) error {
	payloadRaw, ok := (*msg).(map[string]interface{})
	if !ok {
		return errors.New("payload is not a valid map[string]interface{}")
	}
	connectionID, _ := payloadRaw["connection_id"].(string)
	requestingAgentID, _ := payloadRaw["requesting_agent_id"].(string)
	logger.Log.Info("Switching to relay mode",
		"connection_id", connectionID,
		"requesting_agent", requestingAgentID)
	if connectionID != "" && h.Agent.P2PClient != nil {
		h.Agent.P2PClient.CloseConnection(connectionID)
	}
	return nil
}

// HandleRelayModeActivated handles relay mode confirmation
func (h *Handlers) HandleRelayModeActivated(msg *any) error {
	payloadRaw, ok := (*msg).(map[string]interface{})
	if !ok {
		return errors.New("payload is not a valid map[string]interface{}")
	}
	connectionID, _ := payloadRaw["connection_id"].(string)
	logger.Log.Info("Relay mode activated", "connection_id", connectionID)
	return nil
}

// channelReader reads from a channel and implements io.Reader
type channelReader struct {
	dataCh <-chan []byte
	errCh  <-chan error
	buffer []byte
}

// SendFileSystemOverP2P sends file over P2P if connection is available
func (h *Handlers) SendFileSystemOverP2P(connectionID, path string) error {
	if h.Agent.P2PClient == nil {
		return fmt.Errorf("P2P client not initialized")
	}
	conn := h.Agent.P2PClient.GetActiveConnection(connectionID)
	if conn == nil {
		return fmt.Errorf("no active P2P connection for %s", connectionID)
	}
	conn.Mu.RLock()
	status := conn.Status
	conn.Mu.RUnlock()
	if status != "connected" {
		return fmt.Errorf("P2P connection not ready: %s", status)
	}
	dataCh, errCh := h.BusinessService.StreamRequestedFileSystem(path)
	reader := &channelReader{dataCh: dataCh, errCh: errCh}
	if err := h.Agent.P2PClient.SendFileOverP2P(connectionID, reader); err != nil {
		return fmt.Errorf("failed to send file over P2P: %w", err)
	}
	return nil
}

func (r *channelReader) Read(p []byte) (n int, err error) {
	if len(r.buffer) > 0 {
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}
	select {
	case chunk, ok := <-r.dataCh:
		if !ok {
			select {
			case err := <-r.errCh:
				return 0, err
			default:
				return 0, io.EOF
			}
		}
		n = copy(p, chunk)
		if n < len(chunk) {
			r.buffer = chunk[n:]
		}
		return n, nil
	case err := <-r.errCh:
		return 0, err
	}
}
