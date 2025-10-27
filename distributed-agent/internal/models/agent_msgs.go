package models

const (
	AgentMsgHeartbeat = "agent_metrics"
	AgentMsgJobStatus = "agent_job_status"
	AgentConnBreakNotice = "agent_conn_break"
)

type JobStatus struct {
	AgentID string `json:"agent_id"`
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Output  string `json:"output,omitempty"`
}

type Metrics struct {
	AgentID    string      `json:"agent_id"`
	SysMetrics HostMetrics `json:"host_metrics"`
	Timestamp  int64       `json:"timestamp,omitempty"`
}

type ConnBreak struct {
	AgentID    string      `json:"agent_id"`
	Timestamp  int64       `json:"timestamp,omitempty"`
}