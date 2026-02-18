package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/daemon"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/fatih/color"
)

func printBanner() {
	color.Magenta(`
                                           _____   ______   _   _   _______      _____   _        _____                              
                                  /\      / ____| |  ____| | \ | | |__   __|    / ____| | |      |_   _|                             
  ______   ______   ______       /  \    | |  __  | |__    |  \| |    | |      | |      | |        | |      ______   ______   ______ 
 |______| |______| |______|     / /\ \   | | |_ | |  __|   | . ' |    | |      | |      | |        | |     |______| |______| |______|
                               / ____ \  | |__| | | |____  | |\  |    | |      | |____  | |____   _| |_                              
                              /_/    \_\  \_____| |______| |_| \_|    |_|       \_____| |______| |_____|                             
                                                                                                                                     
                                                                                                                                     
`)

}

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
	printBanner()

	agentName, command, err := parseArgs()
	if err != nil {
		printUsageAndExit(err)
	}

	logger.Init("agent.log")

	cfg := config.New(agentName)
	logStartupInfo(cfg)

	businessService := service.NewService()
	manager := daemon.NewApplicationWithManager(cfg, businessService)

	if command != "" {
		handleCLI(manager, command, agentName)
		return
	}

	color.Green("✔ Starting NebulaLink Agent daemon...")
	if err := manager.StartDaemon(); err != nil {
		color.Red("✖ Agent daemon failed: %v", err)
		logger.Log.Error("Agent daemon failed", "err", err)
		os.Exit(1)
	}
}

func parseArgs() (agentName string, command string, err error) {
	if len(os.Args) < 2 {
		return "", "", fmt.Errorf("agent name is required")
	}

	arg1 := strings.ToLower(os.Args[1])

	if arg1 == "-h" || arg1 == "--help" || arg1 == "help" {
		printUsageAndExit(nil)
	}

	if isCLICommand(arg1) {
		if len(os.Args) < 3 {
			return "", "", fmt.Errorf("agent name is required for command: %s", arg1)
		}
		return os.Args[2], arg1, nil
	}

	return os.Args[1], "", nil
}

func isCLICommand(arg string) bool {
	switch arg {
	case "install", "uninstall", "start", "stop":
		return true
	default:
		return false
	}
}

func printUsageAndExit(err error) {
	if err != nil {
		color.Red("✖ Error: %v\n", err)
	}

	color.Cyan("Usage:")
	color.White("  agent <agent-name>")
	color.White("  agent install <agent-name>")
	color.White("  agent uninstall <agent-name>")
	color.White("  agent start <agent-name>")
	color.White("  agent stop <agent-name>")
	color.White("  agent --help")

	color.Cyan("\nExamples:")
	color.Green("  agent branch-agent-01")
	color.Green("  agent install branch-agent-01")
	color.Green("  agent start branch-agent-01")
	color.Green("  agent stop branch-agent-01")
	color.Green("  agent uninstall branch-agent-01\n")

	os.Exit(1)
}

func handleCLI(manager *daemon.DaemonManager, command string, agentName string) {
	color.Yellow("▶ Command: %s", command)
	color.White("🆔 Agent: %s\n", agentName)

	switch command {
	case "install":
		color.Cyan("Installing agent service...")
		must(manager.InstallDaemon())
		color.Green("✔ Service installed successfully!")

	case "uninstall":
		color.Cyan("Uninstalling agent service...")
		must(manager.UninstallDaemon())
		color.Green("✔ Service uninstalled successfully!")

	case "start":
		color.Cyan("Starting agent daemon...")
		must(manager.StartDaemon())
		color.Green("✔ Agent daemon started successfully!")

	case "stop":
		color.Cyan("Stopping agent daemon...")
		must(manager.StopDaemon())
		color.Green("✔ Agent daemon stopped successfully!")

	default:
		color.Red("✖ Unknown command: %s", command)
		log.Fatalf("unknown command: %s", command)
	}
}

func must(err error) {
	if err != nil {
		color.Red("✖ Failed: %v", err)
		log.Fatal(err)
	}
}
