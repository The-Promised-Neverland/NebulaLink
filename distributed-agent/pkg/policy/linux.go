package policy

import (
	"os"
	"path/filepath"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/The-Promised-Neverland/agent/pkg/utils"
)

type LinuxPolicy struct {
	serviceName string
	binaryPath  string
}

func NewLinuxPolicy(cfg *config.Config) *LinuxPolicy {
	return &LinuxPolicy{
		serviceName: cfg.ServiceName(),
		binaryPath:  cfg.BinaryPath(), 
	}
}

func (p *LinuxPolicy) ConfigureAutoStart() error {
	unitPath := filepath.Join(
		"/etc/systemd/system",
		p.serviceName+".service",
	)

	unit := `[Unit]
Description=NebulaLink Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=` + p.binaryPath + `
Restart=always
RestartSec=5
KillSignal=SIGTERM
TimeoutStopSec=30
LimitNOFILE=65536
NoNewPrivileges=true
ProtectSystem=full
ProtectHome=true

[Install]
WantedBy=multi-user.target
`
	if err := os.WriteFile(unitPath, []byte(unit), 0644); err != nil {
		return err
	}
	_, _ = utils.RunCommand("systemctl", "daemon-reexec")
	_, _ = utils.RunCommand("systemctl", "enable", p.serviceName)
	logger.Log.Info("systemd unit installed")
	return nil
}

func (p *LinuxPolicy) ConfigureRestartPolicy() error {
	logger.Log.Info("systemd restart policy enforced via unit")
	return nil
}
