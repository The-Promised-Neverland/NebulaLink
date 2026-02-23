package ws

import (
	"fmt"

	"github.com/The-Promised-Neverland/master-server/internal/models"
)

// RegisterDefaultHandlers registers all default message handlers for the WSHub
func (h *WSHub) RegisterDefaultHandlers() {
	h.RegisterHandler("agent_metrics", func(msg *models.Message, c *Connection) error {
		if payloadMap, ok := msg.Payload.(map[string]interface{}); ok {
			if endpoint, hasEndpoint := payloadMap["public_endpoint"].(string); hasEndpoint && endpoint != "" {
				h.Mutex.Lock()
				c.PublicEndpoint = endpoint
				h.Mutex.Unlock()
				delete(payloadMap, "public_endpoint")
				delete(payloadMap, "nat_type")
				msg.Payload = payloadMap
			}
		}
		return nil
	})

	h.RegisterHandler(models.MasterMsgTransferStatus, func(msg *models.Message, c *Connection) error {
		payloadMap, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return nil
		}
		status, hasStatus := payloadMap["status"].(string)
		if !hasStatus || status == "" {
			return nil
		}
		switch status {
		case "p2p_success":
			connectionID, ok2 := payloadMap["connection_id"].(string)
			if ok2 && connectionID != "" && h.P2PManager != nil {
				h.P2PManager.HandleP2PSuccess(connectionID)
			}
		case "p2p_failed":
			connectionID, ok2 := payloadMap["connection_id"].(string)
			if ok2 && connectionID != "" && h.P2PManager != nil {
				reason := "unknown"
				if r, ok3 := payloadMap["reason"].(string); ok3 {
					reason = r
				}
				h.P2PManager.HandleP2PFailure(connectionID, reason)
			}
		}
		if c.RelayTo != "" {
			statusMsg := models.Message{
				Type: models.MasterMsgTransferStatus,
				Payload: map[string]interface{}{
					"status":   status,
					"agent_id": c.Id, // Source agent (the one sending files)
				},
			}
			if connectionID, ok := payloadMap["connection_id"].(string); ok && connectionID != "" {
				statusMsg.Payload.(map[string]interface{})["connection_id"] = connectionID
			}
			if reason, ok := payloadMap["reason"].(string); ok && reason != "" {
				statusMsg.Payload.(map[string]interface{})["reason"] = reason
			}
			h.Send(c.RelayTo, Outbound{Msg: &statusMsg})
			fmt.Printf("Forwarded '%s' status to destination agent %s from source agent %s\n", status, c.RelayTo, c.Id)
		}
		return nil
	})

	h.RegisterHandler(models.MasterMsgAgentRequestFile, func(msg *models.Message, c *Connection) error {
		payloadMap, ok := msg.Payload.(map[string]interface{})
		if !ok {
			return nil
		}
		requestingAgentID, ok2 := payloadMap["requesting_agent_id"].(string)
		if !ok2 || requestingAgentID == "" {
			return nil
		}
		_, err := h.P2PManager.StartTransfer(requestingAgentID, c.Id)
		if err != nil {
			return err
		}
		return nil
	})

}
