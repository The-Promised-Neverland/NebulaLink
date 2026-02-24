package handlers

import (
	"errors"
	"fmt"
	"path/filepath"

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
	trxfMode, _ := payloadRaw["transfer_mode"].(string)
	logger.Log.Info("Received transfer status", "status", status, "source_agent", sourceAgentID, "mode", trxfMode)
	switch status {
	case "initiated":
		if sourceAgentID == "" {
			return fmt.Errorf("source_agent_id is required to start transfer")
		}
		logger.Log.Info("Transfer initiated - preparing to receive data", "sourceAgent", sourceAgentID, "mode", trxfMode)
		if err := h.TransferManager.Receive(sourceAgentID, trxfMode); err != nil {
			return fmt.Errorf("failed to start receive: %w", err)
		}
		if trxfMode == "relay" {
			h.Agent.BinaryChunkHandler = func(chunk []byte) error {
				return h.TransferManager.WriteChunk(chunk)
			}
		}
		logger.Log.Info("Transfer setup complete, waiting for data")
	case "completed":
		logger.Log.Info("Received 'completed' status - finalizing transfer")
		h.Agent.BinaryChunkHandler = nil
		if err := h.TransferManager.Complete(); err != nil {
			return fmt.Errorf("failed to complete transfer: %w", err)
		}
		logger.Log.Info("Transfer completed and file extracted successfully")
	case "running":
		logger.Log.Info("Transfer in progress", "sourceAgent", sourceAgentID)
	default:
		logger.Log.Info("Transfer status update", "status", status, "sourceAgent", sourceAgentID)
	}

	return nil
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
	trxfMode, _ := payloadRaw["transfer_mode"].(string)
	path = filepath.Clean(path)
	connectionID, _ := payloadRaw["connection_id"].(string)
	logger.Log.Info("[TRANSFER] File transfer request received from master", "filePath", path, "requestInitiator", requestInitiator, "transfer_mode", trxfMode, "connection_id", connectionID)
	if trxfMode == "p2p" {
		logger.Log.Info("[TRANSFER] Master gave green signal - starting P2P file transfer", "connection_id", connectionID, "path", path, "target", requestInitiator)
	} else {
		logger.Log.Info("[TRANSFER] Starting relay file transfer", "path", path, "target", requestInitiator)
	}
	if err := h.TransferManager.Send(path, requestInitiator, trxfMode); err != nil {
		logger.Log.Error("[TRANSFER] Transfer failed, reporting to master", "error", err, "mode", trxfMode, "connection_id", connectionID)
		connectionID, _ := payloadRaw["connection_id"].(string)
		failureMsg := models.Message{
			Type: models.MasterMsgTransferStatus,
			Payload: map[string]interface{}{
				"status":        "transfer_failed",
				"connection_id": connectionID,
				"reason":        err.Error(),
				"agent_id":      h.Config.AgentID(),
			},
		}
		if sendErr := h.Agent.Send(ws.Outbound{Msg: &failureMsg}); sendErr != nil {
			logger.Log.Error("Failed to report transfer failure to master", "error", sendErr)
		}
		return fmt.Errorf("transfer failed: %w", err)
	}
	return nil
}

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
	logger.Log.Info("[P2P] P2P initiation received from master", "connection_id", connectionID, "target_agent", targetAgentID, "target_endpoint", targetEndpoint, "attempt", attemptNumber)
	go func() {
		logger.Log.Info("[P2P] Starting P2P connection attempt", "connection_id", connectionID, "target", targetEndpoint, "attempt", attemptNumber, "countdown", countdownSeconds)
		if err := h.TransferManager.AttemptP2PConnection(
			connectionID,
			targetAgentID,
			targetEndpoint,
			attemptNumber,
			countdownSeconds,
		); err != nil {
			logger.Log.Error("[P2P] P2P connection attempt failed", "connection_id", connectionID, "error", err)
		} else {
			logger.Log.Info("[P2P] P2P connection attempt completed", "connection_id", connectionID)
		}
	}()
	return nil
}

func (h *Handlers) HandleRelayFallback(msg *any) error {
	payloadRaw, ok := (*msg).(map[string]interface{})
	if !ok {
		return errors.New("payload is not a valid map[string]interface{}")
	}
	fallback, _ := payloadRaw["fallback"].(bool)
	if !fallback {
		return fmt.Errorf("fallback flag not set")
	}
	connectionID, _ := payloadRaw["connection_id"].(string)
	logger.Log.Info("Relay fallback received from master", "connection_id", connectionID)
	if connectionID != "" {
		h.TransferManager.CloseP2PConnection(connectionID)
		logger.Log.Info("Closed P2P connection due to relay fallback", "connection_id", connectionID)
	}
	action, _ := payloadRaw["action"].(string)
	if action == "" {
		return fmt.Errorf("action command is missing from relay fallback payload")
	}
	switch action {
	case "send":
		requestingAgentID, _ := payloadRaw["requesting_agent_id"].(string)
		logger.Log.Info("[TRANSFER] Master command: SEND file via relay mode (fallback from P2P)", "action", action, "requesting_agent", requestingAgentID, "connection_id", connectionID)
		return h.SendFileSystem(msg)
	case "receive":
		sourceAgentID, _ := payloadRaw["source_agent_id"].(string)
		logger.Log.Info("[TRANSFER] Master command: RECEIVE file via relay mode (fallback from P2P)", "action", action, "source_agent", sourceAgentID, "connection_id", connectionID)
		if err := h.TransferManager.Receive(sourceAgentID, "relay"); err != nil {
			return fmt.Errorf("failed to prepare receive: %w", err)
		}
		h.Agent.BinaryChunkHandler = func(chunk []byte) error {
			return h.TransferManager.WriteChunk(chunk)
		}
		logger.Log.Info("[TRANSFER] Ready to receive binary chunks via relay", "source_agent", sourceAgentID)
	default:
		return fmt.Errorf("unknown action command: %s", action)
	}

	return nil
}
