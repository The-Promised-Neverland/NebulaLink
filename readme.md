# NebulaLink

A distributed agent management system enabling centralized monitoring and control of remote agents across multiple machines through a hub-and-spoke architecture.

## Live Deployment

**Backend**: http://ec2-16-112-43-203.ap-south-2.compute.amazonaws.com:8081/health  
**Frontend**: http://ec2-16-112-43-203.ap-south-2.compute.amazonaws.com:8080/

> **Note**: This deployment runs on a free-tier AWS account. The EC2 instance is automatically shut down via CloudWatch Events and Lambda to minimize costs. The instance may not be available 24/7.

## System Architecture

NebulaLink implements a **hub-and-spoke architecture** where a central master server coordinates communication between multiple distributed agents and a web-based frontend.

```
                    ┌──────────────┐
                    │ Nebula-Watcher│
                    │   (Frontend)  │
                    └───────┬───────┘
                            │
                    WebSocket + REST
                            │
                    ┌───────▼───────┐
                    │ Master-Server │
                    │     (Hub)     │
                    └───────┬───────┘
                            │
                    WebSocket
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
   ┌────▼────┐         ┌─────▼─────┐      ┌─────▼─────┐
   │ Agent 1 │         │  Agent 2  │      │  Agent N  │
   └─────────┘         └───────────┘      └───────────┘
```

## Backend Architecture

### Master Server

The master server serves as the central coordination hub, managing all WebSocket connections and routing messages between agents and the frontend.

#### Technology Stack

- **Language**: Go 1.23+
- **Web Framework**: Gin (HTTP routing and middleware)
- **WebSocket**: Gorilla WebSocket (WebSocket protocol implementation)
- **Concurrency**: Native Go goroutines and channels
- **Logging**: Structured logging with custom logger

#### Core Packages and Components

**`internal/ws/`** - WebSocket Hub Management
- `ws.go`: Central `Hub` struct managing all WebSocket connections
- `connection.go`: `Connection` struct representing individual WebSocket connections
- `pumps.go`: Read/write pumps for bidirectional message handling
- `ping_pong.go`: Heartbeat mechanism (30s ping interval, 60s pong timeout)

**Architectural Pattern**: The Hub maintains a thread-safe map of connections using `sync.RWMutex` for concurrent access. Each connection runs three goroutines: read pump, write pump, and message processor pump.

**`internal/api/handlers/`** - HTTP and WebSocket Handlers
- `handlers.go`: REST API endpoints for agent queries
- `ws_upgrade.go`: WebSocket upgrade handler, extracts connection metadata from query parameters

**`internal/api/processors/`** - Message Routing
- `ws_mssg_processor.go`: Message processor that routes messages based on source (agent vs frontend)
- Implements a channel-based routing system where messages are queued and dispatched to target connections

**`internal/service/`** - Business Logic Layer
- `service.go`: Service layer providing agent query operations
- Filters and transforms connection data into API responses
- Excludes frontend connections from agent listings

**`internal/api/routers/`** - HTTP Routing
- `router.go`: Gin router configuration, registers all HTTP and WebSocket endpoints

**`internal/models/`** - Domain Models
- `message.go`: WebSocket message structures
- `task.go`: Task definition structures

#### Connection Lifecycle Management

1. **Connection Establishment**: Client connects via WebSocket with `name` and `id` query parameters
2. **Registration**: Connection is registered in the Hub's connection map
3. **Pump Initialization**: Three goroutines start (read, write, processor)
4. **Heartbeat**: Master sends ping every 30 seconds, expects pong within 60 seconds
5. **Reconnection Handling**: Existing connections are reused on reconnection, preventing duplicate entries
6. **Disconnection**: Connection is marked as disconnected (`Conn = nil`) but remains in registry for visibility

#### Message Routing Architecture

The system uses a **processor-based routing pattern**:

```
Agent Message → Processor → Outgoing Channel → Hub → Target Connection
```

- Messages from agents are routed to the frontend
- Messages from frontend can be routed to specific agents (future feature)
- All routing is asynchronous via channels, preventing blocking operations

#### Concurrency Model

- **Goroutines per Connection**: Each WebSocket connection spawns 3 goroutines (read, write, processor)
- **Hub Goroutine**: One goroutine handles message routing from the processor's outgoing channel
- **Mutex Protection**: `sync.RWMutex` protects the connection map for concurrent reads/writes
- **Channel Buffering**: Buffered channels prevent message loss during high load

### Distributed Agent

The distributed agent runs on remote machines, collecting system metrics and monitoring file systems.

#### Technology Stack

- **Language**: Go 1.23+
- **WebSocket Client**: Gorilla WebSocket
- **File Watching**: fsnotify (cross-platform file system notifications)
- **Service Management**: kardianos/service (cross-platform service installation)
- **Configuration**: godotenv (environment variable management)

#### Core Packages and Components

**`internal/ws/`** - WebSocket Client
- `client.go`: WebSocket client implementation with auto-reconnection logic
- `pumps.go`: Read/write pumps for agent-side WebSocket communication

**`internal/agent_worker/`** - Metrics Collection
- `worker.go`: Collects and sends system metrics (CPU, memory, disk, uptime)
- Implements periodic heartbeat mechanism (configurable interval, default 3 seconds)

**`internal/watcher/`** - File System Monitoring
- `watcher.go`: File system watcher using fsnotify
- `events.go`: File event type definitions and filtering
- Implements debouncing (500ms) to handle rapid file changes
- Recursively monitors subdirectories

**`internal/daemon/`** - Service Management
- `application.go`: Main application lifecycle management
- `manager.go`: Cross-platform service installation/uninstallation
- Handles graceful shutdown and reconnection logic

**`internal/config/`** - Configuration Management
- `config.go`: Configuration struct with environment variable loading
- Generates unique agent ID using hash-based algorithm
- Manages service configuration and shared folder paths

**`internal/handlers/`** - Message Handlers
- `handlers.go`: Handles incoming messages from master
- `register.go`: Registers message handlers for different message types

**`pkg/policy/`** - Platform-Specific Policies
- `policy.go`: Policy interface for platform-specific behavior
- `windows.go`, `linux.go`, `darwin.go`: OS-specific implementations

#### Agent Lifecycle

1. **Initialization**: Loads configuration from `.env` or environment variables
2. **Service Installation** (optional): Can install as system service for persistent operation
3. **Connection**: Establishes WebSocket connection to master server
4. **Metrics Loop**: Continuously collects and sends system metrics
5. **File Watching**: Monitors shared directory for file system events
6. **Reconnection**: Automatically reconnects on connection loss with exponential backoff

#### File System Monitoring Architecture

- **Event Detection**: Uses fsnotify to detect create, write, remove, and rename events
- **Debouncing**: 500ms debounce window prevents excessive snapshots from rapid file changes
- **Snapshot Generation**: On file events, scans entire directory tree and sends complete snapshot
- **Initial Snapshot**: Sends directory snapshot immediately upon connection

## Communication Architecture

### WebSocket Protocol

All components communicate via WebSocket connections to the master server. The protocol uses a simple query parameter-based identification system:

**Connection URL Format**:
```
ws://<master-ip>:8081/ws?name=<name>&id=<id>
```

- **Agents**: `name=<agent-name>&id=<agent-id>`
- **Frontend**: `name=frontend&id=frontend`

### Message Flow Patterns

**Agent → Master → Frontend**:
- Agents send metrics and directory snapshots
- Master receives messages via processor
- Processor routes messages to frontend via outgoing channel
- Hub dispatches messages to frontend connection

**Master → Agent**:
- Master sends ping messages every 30 seconds
- Agents respond with pong messages
- Connection timeout if pong not received within 60 seconds

**Frontend → Master**:
- Frontend requests agent list on connection
- Master responds with current agent registry
- Frontend receives real-time updates via WebSocket

### Message Types

- `agent_metrics`: System metrics payload (CPU, memory, disk, uptime, hostname, OS)
- `agent_directory_snapshot`: Complete directory tree structure
- `agent_disconnected`: Disconnection notification
- `agent_list`: Registry of all agents

## Deployment Architecture

### Containerization

The system uses Docker Compose for orchestration:

- **Backend Container**: Multi-stage Go build, Alpine Linux base image
- **Frontend Container**: Node.js build stage + Nginx runtime stage
- **Network**: Bridge network (`app-network`) for inter-service communication
- **Port Mapping**: Frontend (8080:80), Backend (8081:80)

### CI/CD Pipeline

**GitHub Actions Workflow**:
1. Triggers on push to `main` branch
2. Checks out repository
3. Sets up SSH authentication using GitHub Secrets
4. Connects to EC2 instance
5. Pulls latest code from repository
6. Sets environment variables for build
7. Executes `docker-compose up -d --build`

**Infrastructure**:
- EC2 instance hosts Docker Compose deployment
- Elastic IP for consistent access
- CloudWatch Events + Lambda for automated shutdown (cost optimization)

## Frontend Architecture

The frontend is a React + TypeScript application built with Vite, providing a real-time dashboard for monitoring agents.

**Technology Stack**:
- React 18 with TypeScript
- Vite for build tooling
- TanStack Query for data fetching and caching
- WebSocket client for real-time updates
- Tailwind CSS for styling
- Radix UI components

**Key Components**:
- `WebSocketContext`: Manages WebSocket connection and agent state
- `Dashboard`: Grid view of all agents
- `AgentDetail`: Detailed view of individual agent metrics and file structure

The frontend maintains a WebSocket connection to the master server and receives real-time updates for agent metrics, status changes, and directory snapshots.

## License

[Specify license]

