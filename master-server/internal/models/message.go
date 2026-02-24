package models

import "time"

const (
	MasterMsgTransferStatus     = "master_transfer_status"
	MasterMsgTransferIntent     = "master_transfer_intent"
	MasterMsgP2PInitiate        = "master_p2p_initiate"
	MasterMsgP2PStartConnection = "master_p2p_start"
	MasterMsgP2PTransferStart   = "master_p2p_transfer_start"
	MasterMsgRelayTransferStart = "master_relay_transfer_start"
	MasterMsgRelayFallback      = "master_relay_fallback"
)

type Message struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

type HealthCheck struct {
	Status string `json:"sys_status"`
	Uptime int64  `json:"uptime"`
}

type AgentInfo struct {
	AgentID  string    `json:"agent_id"`
	Name     string    `json:"agent_name,omitempty"`
	OS       string    `json:"agent_os"`
	LastSeen time.Time `json:"agent_last_seen"`
}

type Metrics struct {
	AgentID     string  `json:"agent_id"`
	CpuUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	Hostname    string  `json:"hostname"`
	OS          string  `json:"os"`
	Uptime      int64   `json:"uptime"`
	Endpoint    string  `json:"public_endpoint,omitempty"`
}
