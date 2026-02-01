package daemon

import (
	"context"
	"os"
	"path/filepath"
	"time"

	agentworker "github.com/The-Promised-Neverland/agent/internal/agent_worker"
	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/internal/handlers"
	"github.com/The-Promised-Neverland/agent/internal/models"
	"github.com/The-Promised-Neverland/agent/internal/service"
	"github.com/The-Promised-Neverland/agent/internal/watcher"
	"github.com/The-Promised-Neverland/agent/internal/ws"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
)

type Application struct {
	config  *config.Config
	agent   *ws.Agent
	worker  *agentworker.AgentWorker
	service *service.Service
	watcher *watcher.Watcher
}

func newApplication(
	cfg *config.Config,
	svc *service.Service,
) *Application {
	return &Application{
		config:  cfg,
		service: svc,
	}
}

func NewApplicationWithManager(cfg *config.Config, svc *service.Service) (*Application, *DaemonManager) {
	app := newApplication(cfg, svc)
	manager := NewDaemonManager(cfg, app)
	sharedPath, err := cfg.SharedFolderPath()
	if err != nil {
		logger.Log.Warn("Failed to get shared folder path, watcher will not be initialized", "err", err)
	} else {
		filterConfig := watcher.DefaultFilterConfig()
		w, err := watcher.NewWatcher(sharedPath, filterConfig, manager.appCtx)
		if err != nil {
			logger.Log.Warn("Failed to create watcher", "err", err)
		} else {
			if err := w.Start(); err != nil {
				logger.Log.Warn("Failed to start watcher", "err", err)
			} else {
				app.watcher = w
				logger.Log.Info("File watcher initialized", "path", sharedPath)
			}
		}
	}

	return app, manager
}

func (app *Application) Run(appCtx context.Context, daemonManager *DaemonManager) {
	if app.watcher != nil {
		app.startWatcher(appCtx)
	}
	app.superviseConnection(appCtx, daemonManager)
	app.Shutdown()
}

func (app *Application) Shutdown() {
	if app.watcher != nil {
		app.watcher.Stop()
		app.watcher = nil
	}
	if app.agent == nil {
		return
	}
	if app.worker != nil {
		app.worker.SendConnSeverNotice()
	}
	if err := app.agent.Close(); err != nil {
		logger.Log.Error("Error closing agent connection:", "err", err)
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

func (app *Application) superviseConnection(appCtx context.Context, daemonManager *DaemonManager) {
	for {
		select {
		case <-appCtx.Done():
			return
		default:
		}
		app.cleanupAgent()
		app.agent = ws.NewAgent(app.config, appCtx)
		app.worker = agentworker.NewAgentWorker(app.agent, app.service, app.config)
		handlerMgr := handlers.NewHandler(app.agent, app.service, app.config, daemonManager)
		handlerMgr.RegisterHandlers()
		if err := app.agent.Connect(); err != nil {
			logger.Log.Error("Failed to connect to master:", "err", err)
			select {
			case <-appCtx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}
		disconnectCh := app.agent.AgentDisconnected()
		app.agent.RunPumps()
		go app.heartbeatLoop(appCtx, disconnectCh)
		if app.watcher != nil {
			go app.sendInitialDirectorySnapshot()
		}
		select {
		case <-disconnectCh:
		case <-appCtx.Done():
			return
		}
	}
}

func (app *Application) heartbeatLoop(appCtx context.Context, disconnectCh <-chan struct{}) {
	ticker := time.NewTicker(app.config.HeartbeatTimer())
	defer ticker.Stop()
	for {
		select {
		case <-appCtx.Done():
			logger.Log.Info("Stopping heartbeat goroutine for shutdown initiation")
			return
		case <-disconnectCh:
			logger.Log.Info("Stopping heartbeat goroutine due to agent disconnect...")
			return
		case <-ticker.C:
			if err := app.worker.SendHeartbeat(); err != nil {
				logger.Log.Error("Failed to send heartbeat:", "err", err)
			}
		}
	}
}

// startWatcher initializes and starts the file system watcher for the shared folder
func (app *Application) startWatcher(appCtx context.Context) error {
	go app.handleFileEvents(appCtx, app.watcher)
	go app.handleWatcherErrors(appCtx, app.watcher)
	return nil
}

// handleFileEvents processes file system events from the watcher
func (app *Application) handleFileEvents(appCtx context.Context, w *watcher.Watcher) {
	for {
		select {
		case <-appCtx.Done():
			logger.Log.Info("Stopping file event handler")
			return
		case event, ok := <-w.Events():
			if !ok {
				return
			}
			app.processFileEvent(event)
		}
	}
}

// handleWatcherErrors processes errors from the watcher
func (app *Application) handleWatcherErrors(appCtx context.Context, w *watcher.Watcher) {
	for {
		select {
		case <-appCtx.Done():
			return
		case err, ok := <-w.Errors():
			if !ok {
				return
			}
			logger.Log.Error("File watcher error", "err", err)
		}
	}
}

// processFileEvent handles a detected file event
func (app *Application) processFileEvent(event watcher.FileEvent) {
	logger.Log.Info("File event detected",
		"type", event.Type,
		"path", event.Path,
		"timestamp", event.Timestamp,
	)
	if app.agent != nil && app.worker != nil {
		snapshot, err := app.scanDirectory()
		if err != nil {
			logger.Log.Error("Failed to scan directory", "err", err)
			return
		}
		if err := app.worker.SendDirectorySnapshot(snapshot); err != nil {
			logger.Log.Error("Failed to send directory snapshot", "err", err)
		} else {
			logger.Log.Info("Directory snapshot sent", "files", snapshot.Directory.TotalFiles)
		}
	}
}

// scanDirectory recursively scans the shared folder and returns a directory snapshot
func (app *Application) scanDirectory() (models.DirectorySnapshot, error) {
	sharedPath, err := app.config.SharedFolderPath()
	if err != nil {
		return models.DirectorySnapshot{}, err
	}
	var files []models.FileInfo
	var totalSize int64
	err = filepath.Walk(sharedPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Log.Warn("Error accessing path", "path", path, "err", err)
			return nil
		}
		relPath, err := filepath.Rel(sharedPath, path)
		if err != nil {
			relPath = path
		}
		fileInfo := models.FileInfo{
			Name:     info.Name(),
			Path:     relPath,
			Size:     info.Size(),
			Modified: info.ModTime().Format(time.RFC3339),
		}
		if info.IsDir() {
			fileInfo.Type = "directory"
		} else {
			fileInfo.Type = "file"
			totalSize += info.Size()
		}
		files = append(files, fileInfo)
		return nil
	})
	if err != nil {
		return models.DirectorySnapshot{}, err
	}
	fileCount := 0
	for _, f := range files {
		if f.Type == "file" {
			fileCount++
		}
	}
	return models.DirectorySnapshot{
		AgentID:   app.config.AgentID(),
		Timestamp: time.Now().Format(time.RFC3339),
		Directory: models.DirectoryInfo{
			Files:      files,
			TotalFiles: fileCount,
			TotalSize:  totalSize,
		},
	}, nil
}

// sendInitialDirectorySnapshot sends the directory snapshot when agent connects
func (app *Application) sendInitialDirectorySnapshot() {
	time.Sleep(1 * time.Second)
	if app.agent == nil || app.worker == nil {
		return
	}
	snapshot, err := app.scanDirectory()
	if err != nil {
		logger.Log.Error("Failed to scan directory for initial snapshot", "err", err)
		return
	}
	if err := app.worker.SendDirectorySnapshot(snapshot); err != nil {
		logger.Log.Error("Failed to send initial directory snapshot", "err", err)
	} else {
		logger.Log.Info("Initial directory snapshot sent", "files", snapshot.Directory.TotalFiles)
	}
}
