package main

import (
	"fmt"
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
	agentName, err := parseAgentName()
	if err != nil {
		printUsageAndExit(err)
	}
	logger.Init("agent.log")
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

func printUsageAndExit(err error) {
	if err != nil {
		log.Println("Error:", err)
	}

	log.Println(`
Usage:
  agent <agent-name>
  agent install <agent-name>
  agent uninstall <agent-name>
  agent start <agent-name>
  agent stop <agent-name>

Example:
  agent install branch-agent-01
`)
	os.Exit(1)
}

func parseAgentName() (string, error) {
	if len(os.Args) > 1 {
		firstArg := os.Args[1]
		if firstArg == "install" || firstArg == "uninstall" || firstArg == "start" || firstArg == "stop" {
			if len(os.Args) > 2 {
				return os.Args[2], nil
			}
			return "", fmt.Errorf("agent name is required")
		}
		return firstArg, nil
	}
	return "", fmt.Errorf("agent name is required")
}

func handleCLI(manager *daemon.DaemonManager) {
	command := os.Args[1]
	switch command {
	case "install":
		must(manager.Install())
	case "uninstall":
		must(manager.Uninstall())
	case "start":
		must(manager.Run())
	case "stop":
		must(manager.StopService())
	default:
		log.Fatalf("unknown command: %s", command)
	}
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
