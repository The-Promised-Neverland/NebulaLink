package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/The-Promised-Neverland/agent/pkg/idcommands"
	"github.com/joho/godotenv"
)

// Config holds agent configuration. Fields are unexported to prevent modification.
type Config struct {
	agentID            string
	masterServerConn   string
	serviceName        string
	serviceDisplayName string
	serviceDescription string
	heartbeatTimer     time.Duration
	binaryPath         string
}

func defaultPaths() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(
			os.Getenv("ProgramFiles"),
			"NebulaLink",
			"agent.exe",
		)
	case "darwin", "linux":
		return filepath.Join(
			"/usr/local/bin",
			"nebulalink-agent",
		)
	default:
		panic("Cannot set filepath. System unstable...")
	}
}

func New() *Config {
	err := godotenv.Load() // ignore error if .env not found
	if err != nil {
		panic(".env was unable to load. Cannot connect to master...")
	}
	masterURL := os.Getenv("MASTER_URL")
	serviceName := os.Getenv("SERVICE_NAME")
	serviceDisplayName := os.Getenv("SERVICE_DISPLAY_NAME")
	serviceDescription := os.Getenv("SERVICE_DESCRIPTION")
	heartbeatSec, _ := strconv.Atoi(os.Getenv("HEARTBEAT_TIMER"))
	cfg := &Config{
		agentID:            idcommands.GenerateAgentID(),
		masterServerConn:   masterURL,
		serviceName:        serviceName,
		serviceDisplayName: serviceDisplayName,
		serviceDescription: serviceDescription,
		heartbeatTimer:     time.Duration(heartbeatSec) * time.Second,
	}
	cfg.binaryPath = defaultPaths()
	return cfg
}

func (c *Config) AgentID() string {
	return c.agentID
}

func (c *Config) MasterServerConn() string {
	return c.masterServerConn
}

func (c *Config) ServiceName() string {
	return c.serviceName
}

func (c *Config) ServiceDisplayName() string {
	return c.serviceDisplayName
}

func (c *Config) ServiceDescription() string {
	return c.serviceDescription
}

func (c *Config) HeartbeatTimer() time.Duration {
	return c.heartbeatTimer
}

func (c *Config) BinaryPath() string {
	return c.binaryPath
}

// SharedFolderPath returns the OS-specific path for the shared folder on Desktop.
// Windows: C:\Users\<Username>\Desktop\NebulaLink-shared
// Linux: /home/<username>/Desktop/NebulaLink-shared
// macOS: /Users/<username>/Desktop/NebulaLink-shared
func (c *Config) SharedFolderPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, "Desktop", "NebulaLink-shared"), nil
}