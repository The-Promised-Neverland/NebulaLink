package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/control"
	"github.com/The-Promised-Neverland/agent/internal/daemon"
	"github.com/The-Promised-Neverland/agent/internal/osManager"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

func logStartupInfo(cfg *config.Config) {
	logger.Log.Info("🚀 NebulaLink Agent starting",
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
	)

	logger.Log.Info("🆔 Agent identity",
		"agent_id", cfg.AgentID(),
	)
}


func main() {
	logger.Init("agent.log")

	cfg := config.New()
	logStartupInfo(cfg)
	svc := service.NewService()

	var (
		app        *daemon.Application
		manager    *osManager.AgentOSManager
		controller control.AgentController
	)

	// ---------- OS-specific wiring ----------
	switch runtime.GOOS {
	case "windows":
		// Application without controller first
		app = daemon.NewApplication(cfg, svc, nil)
		daemonSvc := daemon.NewAgentDaemon(cfg, app)

		manager = osManager.NewManager(daemonSvc, cfg)
		controller = manager

	case "linux", "darwin":
		app = daemon.NewApplication(cfg, svc, nil)
		manager = osManager.NewManager(nil, cfg)
		controller = manager

	default:
		log.Fatal("unsupported OS")
	}

	// Inject controller AFTER construction (breaks dependency cycles)
	app.SetController(controller)

	// ---------- CLI / INSTALLER ----------
	if len(os.Args) > 1 {
		handleCLI(manager)
		return
	}

	// ---------- RUNTIME ----------
	switch runtime.GOOS {
	case "windows":
		// Windows service runtime (SCM → kardianos)
		if err := manager.Start(); err != nil {
			logger.Log.Error("❌ Windows service failed", "err", err)
		}

	case "linux", "darwin":
		// Native runtime (systemd / launchd)
		shutdownCh := make(chan struct{})
		sigCh := make(chan os.Signal, 1)

		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

		go func() {
			<-sigCh
			logger.Log.Info("🛑 Shutdown signal received")
			close(shutdownCh)
		}()

		app.Run(shutdownCh)
	}
}

// ---------- CLI helpers ----------

func handleCLI(manager *osManager.AgentOSManager) {
	switch os.Args[1] {
	case "install":
		must(manager.Install())
	case "uninstall":
		must(manager.Uninstall())
	case "start":
		must(manager.Start())
	case "stop":
		must(manager.Stop())
	default:
		log.Fatal("unknown command")
	}
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
