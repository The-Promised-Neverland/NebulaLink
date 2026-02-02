package models

const (
	AgentMsgHeartbeat         = "agent_metrics"
	AgentMsgJobStatus         = "agent_job_status"
	AgentConnBreakNotice      = "agent_conn_break"
	AgentMsgDirectorySnapshot = "agent_directory_snapshot"
)

type JobStatus struct {
	AgentID string `json:"agent_id"`
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Output  string `json:"output,omitempty"`
}

type Metrics struct {
	AgentID    string      `json:"agent_id"`
	AgentName  string      `json:"agent_name,omitempty"`
	SysMetrics HostMetrics `json:"host_metrics"`
	Timestamp  int64       `json:"timestamp,omitempty"`
}

type ConnBreak struct {
	AgentID   string `json:"agent_id"`
	Timestamp int64  `json:"timestamp,omitempty"`
}

type DirectorySnapshot struct {
	AgentID   string        `json:"agent_id"`
	Timestamp string        `json:"timestamp"`
	Directory DirectoryInfo `json:"directory"`
}

type DirectoryInfo struct {
	Files      []FileInfo `json:"files"`
	TotalFiles int        `json:"total_files"`
	TotalSize  int64      `json:"total_size"`
}

type FileInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
	Type     string `json:"type"`
}
