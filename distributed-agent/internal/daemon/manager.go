package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/The-Promised-Neverland/agent/pkg/policy"
	kardianos "github.com/kardianos/service"
)

type DaemonManager struct {
	cfg       *config.Config
	app       *Application
	appCtx    context.Context
	appCancel context.CancelFunc
}

func NewDaemonManager(cfg *config.Config, app *Application) *DaemonManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &DaemonManager{
		cfg:       cfg,
		app:       app,
		appCtx:    ctx,
		appCancel: cancel,
	}
}

func (m *DaemonManager) newService() (kardianos.Service, error) {
	if m.app == nil {
		return nil, fmt.Errorf("application cannot be nil")
	}
	return kardianos.New(m, &kardianos.Config{
		Name:        m.cfg.ServiceName(),
		DisplayName: m.cfg.ServiceDisplayName(),
		Description: m.cfg.ServiceDescription(),
	})
}

func (m *DaemonManager) Start(s kardianos.Service) error {
	logger.Log.Info("Kardianos starting service", "service", s.String(), "platform", s.Platform())
	go m.app.Run(m.appCtx, m)
	return nil
}

func (m *DaemonManager) Stop(s kardianos.Service) error {
	logger.Log.Info("Kardianos stopping service", "service", s.String())
	m.appCancel()
	return nil
}

func (m *DaemonManager) InstallDaemon() error {
	if err := m.createSharedFolder(); err != nil {
		return fmt.Errorf("failed to create shared folder: %w", err)
	}
	s, err := m.newService()
	if err != nil {
		return err
	}
	if err := s.Install(); err != nil {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("failed to install Windows service (requires administrator privileges): %w\nPlease run PowerShell or Command Prompt as Administrator", err)
		}
		return fmt.Errorf("failed to install service: %w", err)
	}
	p, err := policy.NewServicePolicy(m.cfg)
	if err != nil {
		return err
	}
	if err := p.ConfigureAutoStart(); err != nil {
		return fmt.Errorf("failed to configure auto-start: %w", err)
	}
	if err := p.ConfigureRestartPolicy(); err != nil {
		return fmt.Errorf("failed to configure restart policy: %w", err)
	}
	return nil
}

func (m *DaemonManager) UninstallDaemon() error {
	s, err := m.newService()
	if err != nil {
		return err
	}
	if err := s.Stop(); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}
	return s.Uninstall()
}

func (m *DaemonManager) RestartDaemon() error {
	s, err := m.newService()
	if err != nil {
		return err
	}
	
	return s.Restart()
}

func (m *DaemonManager) StartDaemon() error {
	s, err := m.newService()
	if err != nil {
		return err
	}
	return s.Run()
}

func (m *DaemonManager) StopDaemon() error {
	s, err := m.newService()
	if err != nil {
		return err
	}
	return s.Stop()
}

func (m *DaemonManager) createSharedFolder() error {
	sharedPath, err := m.cfg.SharedFolderPath()
	if err != nil {
		return err
	}
	if info, err := os.Stat(sharedPath); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("shared folder path exists but is not a directory: %s", sharedPath)
		}
		testFile := filepath.Join(sharedPath, ".test-write")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			return fmt.Errorf("shared folder exists but is not writable: %w", err)
		}
		_ = os.Remove(testFile)
		logger.Log.Info("Shared folder already exists", "path", sharedPath)
		return nil
	}
	var perm os.FileMode = 0755 // rwxr-xr-x for Linux/macOS
	if runtime.GOOS == "windows" {
		perm = 0755 // Windows will use its own permission model
	}
	if err := os.MkdirAll(sharedPath, perm); err != nil {
		return fmt.Errorf("failed to create shared folder: %w", err)
	}
	logger.Log.Info("Created shared folder", "path", sharedPath)
	return nil
}
