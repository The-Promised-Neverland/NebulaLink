package models

const (
	MasterMsgMetricsRequest     = "master_metrics_request"
	MasterMsgTaskAssignment     = "master_task_assigned"
	MasterMsgRestartAgent       = "master_restart_request"
	MasterMsgAgentUninstall     = "master_uninstall_initiated"
	MasterMsgTransferStatus     = "master_transfer_status"
	MasterMsgTransferIntent     = "master_transfer_intent"
	MasterMsgP2PInitiate        = "master_p2p_initiate"
	MasterMsgP2PTransferStart   = "master_p2p_transfer_start"
	MasterMsgRelayTransferStart = "master_relay_transfer_start"
	MasterMsgRelayFallback      = "master_relay_fallback"
)

// TODO: Based on furthur development, shape it up
type TaskAssignmentPayload struct {
	JobID      string `json:"job_id"`
	JobType    string `json:"job_type"`
	Parameters string `json:"parameters"`
}
