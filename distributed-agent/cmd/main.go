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
		"agent_name", cfg.AgentName(),
	)
}

func main() {
	logger.Init("agent.log")
	agentName := parseAgentName()
	cfg := config.New(agentName)
	logStartupInfo(cfg)
	businessService := service.NewService()
	manager := daemon.NewApplicationWithManager(cfg, businessService)
	if len(os.Args) > 1 {
		firstArg := os.Args[1]
		if firstArg == "install" || firstArg == "uninstall" || firstArg == "start" || firstArg == "stop" {
			handleCLI(manager)
			return
		}
	}
	if err := manager.Run(); err != nil {
		logger.Log.Error("Agent daemon failed", "err", err)
	}
}

func parseAgentName() string {
	if len(os.Args) > 1 {
		firstArg := os.Args[1]
		if firstArg == "install" || firstArg == "uninstall" || firstArg == "start" || firstArg == "stop" {
			if len(os.Args) > 2 {
				return os.Args[2]
			}
			return ""
		}
		return firstArg
	}
	return ""
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
