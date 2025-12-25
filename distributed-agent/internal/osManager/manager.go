package osManager

import (
	"fmt"
	"runtime"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/pkg/policy"
	kardianos "github.com/kardianos/service"
)

type AgentOSManager struct {
	daemon kardianos.Interface
	cfg    *config.Config
}

func NewManager(daemon kardianos.Interface, cfg *config.Config) *AgentOSManager {
	return &AgentOSManager{
		daemon: daemon,
		cfg:    cfg,
	}
}

func (m *AgentOSManager) newService() (kardianos.Service, error) {
	return kardianos.New(m.daemon, &kardianos.Config{
		Name:        m.cfg.ServiceName(),
		DisplayName: m.cfg.ServiceDisplayName(),
		Description: m.cfg.ServiceDescription(),
	})
}

func (m *AgentOSManager) Install() error {
	p, err := policy.NewServicePolicy(m.cfg)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		s, err := m.newService()
		if err != nil {
			return err
		}
		if err := s.Install(); err != nil {
			return err
		}
		_ = p.ConfigureAutoStart()
		_ = p.ConfigureRestartPolicy()
		return s.Start()
	case "linux", "darwin":
		if err := p.ConfigureAutoStart(); err != nil {
			return err
		}
		if err := p.ConfigureRestartPolicy(); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func (m *AgentOSManager) Uninstall() error {
	s, err := m.newService()
	if err != nil {
		return err
	}
	_ = s.Stop()
	return s.Uninstall()
}

func (m *AgentOSManager) Restart() error {
	s, err := m.newService()
	if err != nil {
		return err
	}
	return s.Restart()
}

func (m *AgentOSManager) Start() error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("Run() is Windows-only")
	}
	s, err := m.newService()
	if err != nil {
		return err
	}
	return s.Run()
}

func (m *AgentOSManager) Stop() error {
	s, err := m.newService()
	if err != nil {
		return err
	}
	return s.Stop()
}
