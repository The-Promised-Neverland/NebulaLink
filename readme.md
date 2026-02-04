# NebulaLink

A distributed agent management system enabling centralized monitoring and control of remote agents through a hub-and-spoke architecture.

## Live Deployment

**Backend**: http://ec2-16-112-43-203.ap-south-2.compute.amazonaws.com:8081/health  
**Frontend**: http://ec2-16-112-43-203.ap-south-2.compute.amazonaws.com:8080/

> **Note**: Running on free-tier AWS. EC2 instance auto-shuts down via CloudWatch + Lambda to minimize costs. May not be available 24/7.

## Architecture Overview

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

**Tech Stack**: Go 1.23+, Gin, Gorilla WebSocket

**Core Components**:
- `internal/ws/`: Hub managing all WebSocket connections with thread-safe map (`sync.RWMutex`)
- `internal/api/processors/`: Message routing processor (agent → frontend, frontend → agent)
- `internal/service/`: Business logic layer for agent queries

**Connection Management**:
- Each connection runs 3 goroutines: read pump, write pump, processor pump
- Heartbeat: 30s ping, 60s pong timeout
- Reconnection reuses existing connection entries

### WebSocket Pump Architecture

**Master Server Pumps** (per connection):
```
ReadPump → IncomingCh → ProcessorPump → OutgoingCh → Hub → WritePump
   ↓                                                              ↓
WebSocket                                                    WebSocket
```

**Agent Pumps** (per agent):
```
ReadPump → IncomingCh → DispatchPump → Handlers
   ↓
WebSocket

SendCh → WritePump → WebSocket
```

**Pump Responsibilities**:
- **ReadPump**: Continuously reads from WebSocket, unmarshals JSON, pushes to incoming channel
- **WritePump**: Reads from send channel, marshals JSON, writes to WebSocket (includes ping ticker)
- **ProcessorPump** (master): Routes messages based on source to appropriate targets
- **DispatchPump** (agent): Routes incoming messages to registered handlers

### Distributed Agent

**Tech Stack**: Go 1.23+, Gorilla WebSocket, fsnotify, kardianos/service

**Core Components**:
- `internal/ws/`: WebSocket client with auto-reconnection
- `internal/agent_worker/`: Metrics collection (CPU, memory, disk, uptime)
- `internal/watcher/`: File system monitoring with debouncing (500ms)
- `internal/daemon/`: Cross-platform service management

**Agent Lifecycle**:
1. Loads config from `.env`
2. Connects to master via WebSocket
3. Starts metrics loop (3s interval)
4. Monitors shared folder for file changes
5. Auto-reconnects on disconnect

## File Sharing Architecture

### Current Implementation

Each agent monitors a **shared folder** (default: `NebulaLink-shared` in home directory):

```
Agent 1                    Agent 2                    Agent N
   │                          │                          │
   └─── Shared Folder ───────┴─── Shared Folder ───────┘
        (monitored)              (monitored)              (monitored)
              │                          │                          │
              └─────────── Directory Snapshots ────────────┘
                              │
                    Master Server (Hub)
                              │
                    Frontend (Display)
```

**File Monitoring Flow**:
1. Agent watches shared folder using `fsnotify`
2. On file event (create/write/remove/rename), debounces for 500ms
3. Scans entire directory tree
4. Sends complete directory snapshot to master
5. Master routes snapshot to frontend for display

**Why Directory Snapshots?**
- Foundation for future **agent-to-agent file access**
- Agents can request files from other agents' shared folders
- Master server will route file requests between agents
- Enables distributed file sharing across the agent network

### Future File Sharing (Planned)

```
Agent 1 requests file from Agent 2:
Agent 1 → Master → Agent 2 → Master → Agent 1
```

The directory snapshots displayed in the frontend are the foundation for this cross-agent file access feature.

## Message Flow

**Agent → Master → Frontend**:
```
Agent sends metrics/snapshot
    ↓
Master ReadPump receives
    ↓
ProcessorPump routes to frontend
    ↓
Frontend receives via WebSocket
```

**Master → Agent**:
```
Master WritePump sends ping (every 30s)
    ↓
Agent ReadPump receives
    ↓
Agent responds with pong
```

## Scalability Estimates

### Current Architecture Limits

**Single Master Server**:
- **Theoretical**: ~10,000 concurrent WebSocket connections (Go goroutines are lightweight)
- **Practical**: ~1,000-2,000 agents (considering message throughput)

**Bottlenecks**:
- In-memory connection map (O(1) lookup, but memory bound)
- Single message processor (can be parallelized)
- No persistence (connections lost on restart)

**Message Throughput**:
- Each agent sends metrics every 3s = ~333 messages/second for 1000 agents
- Directory snapshots are larger but infrequent (on file changes)
- Go's channel-based architecture handles this efficiently

### Scaling Path

**To 10,000+ agents**:
- **Master clustering**: Multiple master instances with load balancing
- **Database backend**: Redis for connection registry, PostgreSQL for metrics history
- **Message queue**: RabbitMQ/Kafka for message routing at scale
- **Agent groups**: Shard agents across master instances

**Estimated Resource Requirements** (10,000 agents):
- Master server: 4-8 CPU cores, 16-32GB RAM
- Network: 100+ Mbps for message throughput
- Storage: Database for metrics persistence

## Deployment

**Docker Compose**: Two services (backend, frontend) on bridge network

**CI/CD**: GitHub Actions → EC2 via SSH
- Auto-deploys on push to `main`
- Uses GitHub Secrets for SSH keys
- CloudWatch + Lambda for cost optimization (auto-shutdown)

**Ports**: Frontend (8080), Backend (8081)

## Frontend

React + TypeScript + Vite. Real-time dashboard showing agent metrics, file structures, and status. WebSocket connection to master for live updates.

## License

[Specify license]
