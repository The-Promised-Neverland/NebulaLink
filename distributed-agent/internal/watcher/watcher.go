package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/fsnotify/fsnotify"
)

// Watcher monitors a directory for file system changes
type Watcher struct {
	watchPath     string
	filterConfig  FilterConfig
	events        chan FileEvent
	errors        chan error
	fsWatcher     *fsnotify.Watcher
	debounceMap   map[string]*time.Timer
	debounceMu    sync.Mutex
	debounceDelay time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewWatcher creates a new file system watcher
func NewWatcher(watchPath string, filterConfig FilterConfig, appCtx context.Context) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(appCtx)
	return &Watcher{
		watchPath:     watchPath,
		filterConfig:  filterConfig,
		events:        make(chan FileEvent, 100),
		errors:        make(chan error, 10),
		fsWatcher:     fsWatcher,
		debounceMap:   make(map[string]*time.Timer),
		debounceDelay: 500 * time.Millisecond, // Debounce rapid events
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start begins watching the directory
func (w *Watcher) Start() error {
	if err := w.fsWatcher.Add(w.watchPath); err != nil {
		return err
	}
	if w.filterConfig.WatchSubdirectories {
		if err := w.addSubdirectories(w.watchPath); err != nil {
			logger.Log.Warn("Failed to add some subdirectories", "err", err)
		}
	}
	logger.Log.Info("File watcher started", "path", w.watchPath)
	w.wg.Add(2)
	go w.eventLoop()
	go w.errorLoop()
	return nil
}

// Stop stops the watcher and cleans up resources
func (w *Watcher) Stop() {
	w.cancel()
	w.fsWatcher.Close()
	w.wg.Wait()
	w.debounceMu.Lock()
	for _, timer := range w.debounceMap {
		timer.Stop()
	}
	w.debounceMap = nil
	w.debounceMu.Unlock()
	close(w.events)
	close(w.errors)
	logger.Log.Info("File watcher stopped")
}

// Events returns the channel of file events
func (w *Watcher) Events() <-chan FileEvent {
	return w.events
}

// Errors returns the channel of errors
func (w *Watcher) Errors() <-chan error {
	return w.errors
}

func (w *Watcher) eventLoop() {
	defer w.wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		}
	}
}

func (w *Watcher) errorLoop() {
	defer w.wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			return
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			select {
			case w.errors <- err:
			default:
				logger.Log.Error("Error channel full, dropping error", "err", err)
			}
		}
	}
}

// handleEvent processes a single fsnotify event
func (w *Watcher) handleEvent(event fsnotify.Event) {
	if !w.filterConfig.ShouldProcess(event.Name) {
		return
	}
	var eventType EventType
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		eventType = EventCreate
		if w.filterConfig.WatchSubdirectories {
			if err := w.fsWatcher.Add(event.Name); err != nil {
				logger.Log.Warn("Failed to watch new subdirectory", "path", event.Name, "err", err)
			}
		}
	case event.Op&fsnotify.Write == fsnotify.Write:
		eventType = EventWrite
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		eventType = EventRemove
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = EventRename
	default:
		return
	}
	w.debounceEvent(eventType, event.Name)
}

// debounceEvent debounces rapid events for the same file
func (w *Watcher) debounceEvent(eventType EventType, filePath string) {
	w.debounceMu.Lock()
	defer w.debounceMu.Unlock()
	if timer, exists := w.debounceMap[filePath]; exists {
		timer.Stop()
	}
	timer := time.AfterFunc(w.debounceDelay, func() {
		w.debounceMu.Lock()
		delete(w.debounceMap, filePath)
		w.debounceMu.Unlock()
		fileEvent := FileEvent{
			Type:      eventType,
			Path:      filePath,
			Timestamp: time.Now(),
		}
		select {
		case w.events <- fileEvent:
		case <-w.ctx.Done():
		default:
			logger.Log.Warn("Events channel full, dropping event", "path", filePath)
		}
	})
	w.debounceMap[filePath] = timer
}

// addSubdirectories recursively adds subdirectories to the watcher
func (w *Watcher) addSubdirectories(rootPath string) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on error
		}
		if path == "transfers" { // no need to listen on transfers
			return nil
		}
		if info.IsDir() {
			if err := w.fsWatcher.Add(path); err != nil {
				logger.Log.Warn("Failed to watch subdirectory", "path", path, "err", err)
			}
		}
		return nil
	})
}
