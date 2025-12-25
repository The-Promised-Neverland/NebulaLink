package policy

import (
	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/The-Promised-Neverland/agent/pkg/utils"
)

type WindowsPolicy struct {
	serviceName string
}

func NewWindowsPolicy(cfg *config.Config) *WindowsPolicy {
	return &WindowsPolicy{
		serviceName: cfg.ServiceName(),
	}
}
func (p *WindowsPolicy) ConfigureAutoStart() error {
	_, err := utils.RunCommand(
		"sc", "config", p.serviceName, "start=", "auto",
	)
	if err != nil {
		logger.Log.Warn("⚠️ Failed to configure Windows auto-start", "err", err)
		return err
	}
	logger.Log.Info("✅ Windows auto-start configured")
	return nil
}


func (p *WindowsPolicy) ConfigureRestartPolicy() error {
	_, err := utils.RunCommand(
		"sc", "failure", p.serviceName,
		"actions=restart/5000/restart/5000/restart/5000",
		"reset=86400",
	)
	if err != nil {
		logger.Log.Warn("⚠️ Failed to configure Windows restart policy", "err", err)
		return err
	}
	logger.Log.Info("🔁 Windows restart policy configured")
	return nil
}
