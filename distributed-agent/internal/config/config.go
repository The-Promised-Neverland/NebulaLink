package config

import (
	"os"
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
	heartbeatTimer     time.Duration // stored as duration for convenience
}

func New() *Config {
	_ = godotenv.Load() // ignore error if .env not found

	masterURL := os.Getenv("MASTER_URL")

	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = "PromisedNeverland"
	}

	serviceDisplayName := os.Getenv("SERVICE_DISPLAY_NAME")
	if serviceDisplayName == "" {
		serviceDisplayName = "Promised Neverland Agent"
	}

	serviceDescription := os.Getenv("SERVICE_DESCRIPTION")
	if serviceDescription == "" {
		serviceDescription = "Remote agent that connects to master server for task execution and system monitoring"
	}

	heartbeatSec, err := strconv.Atoi(os.Getenv("HEARTBEAT_TIMER"))
	if err != nil || heartbeatSec <= 0 {
		heartbeatSec = 10
	}

	return &Config{
		agentID:            idcommands.GenerateAgentID(),
		masterServerConn:   masterURL,
		serviceName:        serviceName,
		serviceDisplayName: serviceDisplayName,
		serviceDescription: serviceDescription,
		heartbeatTimer:     time.Duration(heartbeatSec) * time.Second,
	}
}

// Getter methods (immutable from outside)

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
