package models

const (
	MasterMsgMetricsRequest   = "master_metrics_request"
	MasterMsgTaskAssignment   = "master_task_assigned"
	MasterMsgRestartAgent     = "master_restart_request"
	MasterMsgAgentUninstall   = "master_uninstall_initiated"
	MasterMsgRelayManager     = "master_filetransfer_manager"
	MasterMsgAgentRequestFile = "master_coordinated_transfer_request"
)

// TODO: Based on furthur development, shape it up
type TaskAssignmentPayload struct {
	JobID      string `json:"job_id"`
	JobType    string `json:"job_type"`
	Parameters string `json:"parameters"`
}
