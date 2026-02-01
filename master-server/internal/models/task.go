package models

import "time"

type Task struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // "shell_command", "python_script", etc.
	Command     string    `json:"command"`     // The actual command/script to run
	AgentID     string    `json:"agent_id"`    // Target agent (empty = any available)
	Status      string    `json:"status"`      // "pending", "running", "completed", "failed"
	CreatedAt   time.Time `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Output      string    `json:"output,omitempty"`
	Error       string    `json:"error,omitempty"`
	Timeout     int       `json:"timeout"` // Timeout in seconds
}

type CreateTaskRequest struct {
	Type    string `json:"type" binding:"required"`    // "shell_command"
	Command string `json:"command" binding:"required"` // "echo hello" or "python script.py"
	AgentID string `json:"agent_id,omitempty"`         // Optional: target specific agent
	Timeout int    `json:"timeout,omitempty"`         // Optional: timeout in seconds (default: 30)
}

type TaskResponse struct {
	Task *Task `json:"task"`
}

type TaskListResponse struct {
	Tasks []*Task `json:"tasks"`
	Total int     `json:"total"`
}




