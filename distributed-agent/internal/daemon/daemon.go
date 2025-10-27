package daemon

import (
	"github.com/The-Promised-Neverland/agent/internal/config"
	kardianos "github.com/kardianos/service"
)

type AgentDaemon struct {
	config *config.Config
	app    *Application
	shutdownCh chan struct{}
}

func NewAgentDaemon(cfg *config.Config, app *Application) *AgentDaemon {
	return &AgentDaemon{
		config: cfg,
		app:    app,
		shutdownCh: make(chan struct{}),
	}
}

// kardianos.Interface implementation
func (d *AgentDaemon) Start(s kardianos.Service) error {
	go d.app.Run(d.shutdownCh)
	return nil
}

func (d *AgentDaemon) Stop(s kardianos.Service) error {
	close(d.shutdownCh)
	d.app.Shutdown()
	return nil
}
