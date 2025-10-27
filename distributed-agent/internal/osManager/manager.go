package osManager

import (
	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
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
	s, err := m.newService()
	if err != nil {
		return err
	}
	if err := s.Install(); err != nil {
		return err
	}
	policy.ConfigureAutoStart()
	policy.ConfigureRestartPolicy()
	if err := s.Start(); err != nil {
		logger.Log.Error("❌ Failed to start service after install:", "err", err)
		return err
	}
	return nil
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

func (m *AgentOSManager) Run() error {
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