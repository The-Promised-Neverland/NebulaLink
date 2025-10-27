package models

import "time"

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
}
