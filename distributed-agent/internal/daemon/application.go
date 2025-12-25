package daemon

import (
	"time"

	agentworker "github.com/The-Promised-Neverland/agent/internal/agent_worker"
	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/control"
	"github.com/The-Promised-Neverland/agent/internal/handlers"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/internal/ws"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

type Application struct {
	config     *config.Config
	agent      *ws.Agent
	worker     *agentworker.AgentWorker
	service    *service.Service
	controller control.AgentController
}

func NewApplication(
	cfg *config.Config,
	svc *service.Service,
	ctrl control.AgentController,
) *Application {
	return &Application{
		config:     cfg,
		service:    svc,
		controller: ctrl,
	}
}

func (app *Application) Run(shutdownCh chan struct{}) {
	go app.superviseConnection(shutdownCh)
	<-shutdownCh
	app.Shutdown()
}

func (app *Application) Shutdown() {
	if app.agent == nil {
		return
	}
	if app.worker != nil {
		app.worker.SendConnSeverNotice()
	}
	if err := app.agent.Close(); err != nil {
		logger.Log.Error("⚠️ Error closing agent connection:", "err", err)
	}
	app.agent = nil
	app.worker = nil
}

func (app *Application) cleanupAgent() {
	if app.agent != nil {
		app.worker.SendConnSeverNotice()
		_ = app.agent.Close()
		app.agent = nil
		app.worker = nil
	}
}

func (app *Application) superviseConnection(shutdownCh chan struct{}) {
	for {
		select {
		case <-shutdownCh:
			return
		default:
		}

		app.cleanupAgent()

		app.agent = ws.NewAgent(app.config)
		app.worker = agentworker.NewAgentWorker(app.agent, app.service, app.config)

		handlerMgr := handlers.NewHandler(app.agent, app.service, app.config, app.controller)
		handlerMgr.RegisterHandlers()

		if err := app.agent.Connect(); err != nil {
			logger.Log.Error("❌ Failed to connect to master:", "err", err)
			time.Sleep(5 * time.Second)
			continue
		}
		disconnectCh := app.agent.AgentDisconnected()
		app.agent.RunPumps()
		go app.heartbeatLoop(shutdownCh, disconnectCh)
		<-disconnectCh
	}
}

func (app *Application) heartbeatLoop(shutdownCh chan struct{}, disconnectCh <-chan struct{}) {
	ticker := time.NewTicker(app.config.HeartbeatTimer())
	defer ticker.Stop()
	for {
		select {
		case <-shutdownCh:
			logger.Log.Info("Stopping heartbeat goroutine for shutdown initiation")
			return
		case <-disconnectCh:
			logger.Log.Info("Stopping heartbeat goroutine due to agent disconnect...")
			return
		case <-ticker.C:
			if err := app.worker.SendHeartbeat(); err != nil {
				logger.Log.Error("⚠️ Failed to send heartbeat:", "err", err)
			}
		}
	}
}

func (app *Application) SetController(c control.AgentController) {
	app.controller = c
}
