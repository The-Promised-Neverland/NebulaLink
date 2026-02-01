package main

import (
	"log"
	"os"
	"runtime"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/daemon"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

func logStartupInfo(cfg *config.Config) {
	logger.Log.Info("NebulaLink Agent starting",
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
	)
	logger.Log.Info("Agent identity",
		"agent_id", cfg.AgentID(),
	)
}


func main() {
	logger.Init("agent.log")
	cfg := config.New()
	logStartupInfo(cfg)
	businessService := service.NewService()
	_, manager := daemon.NewApplicationWithManager(cfg, businessService)
	if len(os.Args) > 1 {
		handleCLI(manager)
		return
	}
	if err := manager.Run(); err != nil {
		logger.Log.Error("Agent daemon failed", "err", err)
	}
}

func handleCLI(manager *daemon.DaemonManager) {
	switch os.Args[1] {
	case "install":
		must(manager.Install())
	case "uninstall":
		must(manager.Uninstall())
	case "start":
		must(manager.Run())
	case "stop":
		must(manager.StopService())
	default:
		log.Fatal("unknown command")
	}
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
