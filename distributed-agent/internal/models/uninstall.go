package models

type UninstallReason struct {
	AgentID   string `json:"agent_id"`
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}