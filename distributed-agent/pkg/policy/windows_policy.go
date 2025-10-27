package policy

import (
	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/The-Promised-Neverland/agent/pkg/utils"
)

// configureRestartPolicy sets up Windows restart policy
func ConfigureRestartPolicy() {
	_, err := utils.RunCommand("sc", "failure", config.New().ServiceName(),
		"actions=restart/5000/restart/5000/restart/5000", "reset=86400")
	if err != nil {
		logger.Log.Warn("⚠️ Failed to set restart policy:", "warn", err)
	} else {
		logger.Log.Info("🔁 Restart policy configured.")
	}
}

// configureAutoStart sets up Windows auto-start
func ConfigureAutoStart() {
	_, err := utils.RunCommand("sc", "config", config.New().ServiceName(), "start=", "auto")
	if err != nil {
		logger.Log.Warn("⚠️ Failed to configure auto-start:", "warn", err)
	} else {
		logger.Log.Info("✅ Auto-start configured.")
	}
}
