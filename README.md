# Master-Agent Monitoring System

A distributed, cross-platform monitoring system built in Go that enables centralized management and monitoring of multiple agent daemons across Windows, Linux, and macOS environments.

## Architecture Overview

<img width="1568" height="555" alt="image" src="https://github.com/user-attachments/assets/2703ac74-47e4-4252-88f0-bf2e3705934d" />


## 🏗️ System Architecture

### Master Server
The central control plane responsible for:
- **Real-time Agent Monitoring**: Tracks agent health, system metrics, and connection status
- **WebSocket Hub**: Maintains persistent bidirectional connections with all agents
- **REST API**: Handles callback responses and administrative operations
- **In-Memory State**: Maintains current agent status and metrics in memory

### Agent Daemons
Lightweight, cross-platform daemons that run persistently on host machines:
- **System Metrics Collection**: CPU usage, disk space, memory, network stats, uptime, OS information
- **Heartbeat Mechanism**: Sends health signals every 5 minutes to confirm operational status
- **Local Log Files**: Maintains rotated logs locally for diagnosing failures and downtime events
- **Bidirectional Communication**: Receives commands via WebSocket, responds via REST callbacks
- **Process Persistence**: Managed by Kardianos service wrapper to ensure daemon survives system operations

## 🔄 Communication Flow

### 1. **Agent → Master (WebSocket)**
- Initial connection and registration
- Periodic heartbeat (every 5 minutes)
- System metrics streaming
- Real-time status updates

### 2. **Master → Agent (WebSocket)**
- On-demand status requests (callback triggers)
- Log file retrieval requests
- Configuration updates
- Command execution requests
- Remote control signals

### 3. **Agent → Master (REST API)**
- Callback responses (when master requests immediate status before next heartbeat)
- Log file uploads (upon server request)
- Large payload transfers

## 🚀 Features

### Current Implementation
- ✅ Cross-platform agent support (Windows, Linux, macOS)
- ✅ WebSocket-based real-time communication
- ✅ Daemon process management via Kardianos
- ✅ System metrics collection and reporting
- ✅ Heartbeat monitoring (5-minute intervals)
- ✅ Local log rotation on agents
- ✅ Callback mechanism for on-demand queries
- ✅ Manual installation and uninstallation
- ✅ In-memory state management

### Roadmap
- 🔲 Authentication and authorization layer
- 🔲 Remote agent installation/uninstallation
- 🔲 Encrypted WebSocket connections (WSS)
- 🔲 Agent auto-discovery and registration
- 🔲 Alert system for agent failures
- 🔲 Dashboard UI for visual monitoring
- 🔲 Agent command execution framework
- 🔲 Multi-tenancy support
- 🔲 Persistent storage layer (database)

## 📋 Prerequisites

- Go 1.21 or higher
- Supported OS: Windows, Linux, macOS
- Network connectivity between agents and master server

## 🛠️ Installation

### Master Server
```bash
# Clone repository
git clone <your-repo>
cd master-agent

# Build master
cd master
go build -o master

# Run master
./master
```

### Agent Daemon
```bash
# Build agent
cd agent
go build -o agent

# Install as service (requires root/admin privileges)
sudo ./agent install

# Start service
sudo ./agent start
```

## 🔧 Configuration

### Master Configuration
```yaml
# config/master.yaml
server:
  host: "0.0.0.0"
  port: 8080
  ws_path: "/ws"

heartbeat:
  timeout: 600 # seconds (10 minutes)
  cleanup_interval: 300 # seconds
```

### Agent Configuration
```yaml
# config/agent.yaml
master:
  url: "ws://master-server:8080/ws"
  api_url: "http://master-server:8080/api"

metrics:
  interval: 300 # seconds (5 minutes)
  
logging:
  level: "info"
  rotation_size: 10 # MB
  max_backups: 5
```

## 📊 Callback Mechanism

The callback system enables the master to request immediate agent status outside the regular heartbeat cycle:

1. **Master** sends callback request via WebSocket
2. **Agent** receives request, gathers current metrics
3. **Agent** POSTs response to Master's REST API endpoint
4. **Master** processes and updates in-memory state

This allows for:
- Immediate health checks on-demand
- Quick response to administrative queries
- Reduced latency for critical operations

## 📝 Logging & Diagnostics

### Agent-Side Log Management
Each agent maintains its own local log files with automatic rotation:

- **Local Storage**: Logs are written to disk on the agent's host machine
- **Log Rotation**: Automatic rotation based on file size/age to prevent disk overflow
- **On-Demand Upload**: Master server can request log files via WebSocket
- **Transmission**: Agent sends requested log files back to master via REST POST

This architecture ensures:
- ✅ Minimal network overhead (logs only sent when needed)
- ✅ Local debugging capability even when disconnected
- ✅ Centralized log analysis when required by admin
- ✅ Efficient storage management on agent hosts

### Log Retrieval Flow
1. **Admin** requests logs for specific agent via Master
2. **Master** sends log request to agent via WebSocket
3. **Agent** reads requested log file from local disk
4. **Agent** POSTs log file to Master's REST API endpoint
5. **Master** serves logs to admin for analysis

## 🔍 Failure Analysis

When an agent experiences downtime or failures:
- Events are captured in local rotated log files
- Master detects missing heartbeats and flags the agent
- Admin can request log files from the agent (if it comes back online)
- Logs help diagnose root causes: network issues, system crashes, resource exhaustion, etc.
- For permanent failures, logs remain on agent host for manual retrieval

## 📝 Manual Uninstallation

### Windows
```cmd
agent.exe stop
agent.exe uninstall
```

### Linux/macOS
```bash
sudo ./agent stop
sudo ./agent uninstall
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

[Your License Here]

---

**Note**: This is an active development project. Authentication, persistent storage, and remote management features are planned for upcoming releases.
