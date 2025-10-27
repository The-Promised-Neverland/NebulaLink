package main

import (
	"os"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/daemon"
	"github.com/The-Promised-Neverland/agent/internal/osManager"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

func main() {
	logger.Init("agent.log")
	cfg := config.New()
	placeholderDaemon := daemon.NewAgentDaemon(cfg, nil)
	manager := osManager.NewManager(placeholderDaemon, cfg)
	svc := service.NewService(manager)
	app := daemon.NewApplication(cfg, svc)
	agentDaemon := daemon.NewAgentDaemon(cfg, app)
	manager = osManager.NewManager(agentDaemon, cfg)

	if len(os.Args) > 1 && os.Args[1] == "install" {
		if err := manager.Install(); err != nil {
			logger.Log.Error("❌ Install failed", "err", err)
		}
		logger.Log.Info("✅ Service installed")
		return
	}

	// DEBUGGER
	if err := manager.Run(); err != nil {
		logger.Log.Error("❌ Service failed", "err", err)
	}
}
