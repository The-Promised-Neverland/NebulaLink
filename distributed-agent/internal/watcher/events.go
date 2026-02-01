package watcher

import (
	"time"
)

type EventType string

const (
	EventCreate EventType = "create"
	EventWrite  EventType = "write"
	EventRemove EventType = "remove"
	EventRename EventType = "rename"
)

type FileEvent struct {
	Type      EventType
	Path      string
	Timestamp time.Time
}

// FilterConfig configures which files to watch
type FilterConfig struct {
	AllowedExtensions []string
	IgnorePatterns []string
	WatchSubdirectories bool
}

// DefaultFilterConfig returns a default filter configuration
func DefaultFilterConfig() FilterConfig {
	return FilterConfig{
		AllowedExtensions:   []string{}, // Empty means allow all file types
		IgnorePatterns:      []string{".tmp", ".swp", ".DS_Store", "~"},
		WatchSubdirectories: true,
	}
}

// ShouldProcess checks if a file should be processed based on filter config
func (fc *FilterConfig) ShouldProcess(filePath string) bool {
	if len(fc.AllowedExtensions) > 0 {
		extMatched := false
		for _, ext := range fc.AllowedExtensions {
			if len(filePath) >= len(ext) && filePath[len(filePath)-len(ext):] == ext {
				extMatched = true
				break
			}
		}
		if !extMatched {
			return false
		}
	}
	for _, pattern := range fc.IgnorePatterns {
		if len(filePath) >= len(pattern) {
			if filePath[len(filePath)-len(pattern):] == pattern {
				return false
			}
		}
	}
	return true
}

