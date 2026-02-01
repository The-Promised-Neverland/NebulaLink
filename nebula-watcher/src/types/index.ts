// NebulaLink DTOs - Complete TypeScript Definitions

// Base Message
export interface Message<T = unknown> {
  type: string;
  payload?: T;
}

// Agent Information
export interface AgentInfo {
  agent_id: string;           // Format: "agent:123"
  agent_os: string;           // "Windows", "Linux", "darwin"
  agent_last_seen: string;    // ISO 8601 timestamp
}

export interface AgentListResponse extends Message<AgentInfo[]> {
  type: "agent_list";
}

export interface AgentInfoResponse extends Message<AgentInfo> {
  type: "agent_info";
}

// Host Metrics
export interface HostMetrics {
  cpu_usage: number;          // Float 0-100
  memory_usage: number;       // Float 0-100
  disk_usage: number;         // Float 0-100
  hostname: string;
  os: string;
  uptime: number;             // Uint64 (seconds)
}

export interface MetricsPayload {
  agent_id: string;
  host_metrics: HostMetrics;
  timestamp?: number;         // Unix timestamp
}

export interface MetricsMessage extends Message<MetricsPayload> {
  type: "agent_metrics";
}

// Directory Snapshot
export interface FileInfo {
  name: string;
  path: string;               // Relative path
  size: number;               // Int64 (bytes)
  modified: string;           // ISO 8601 timestamp
  type: "file" | "directory";
}

export interface DirectoryInfo {
  files: FileInfo[];
  total_files: number;
  total_size: number;         // Int64 (bytes)
}

export interface DirectorySnapshot {
  agent_id: string;
  timestamp: string;          // ISO 8601 timestamp
  directory: DirectoryInfo;
}

export interface DirectorySnapshotMessage extends Message<DirectorySnapshot> {
  type: "agent_directory_snapshot";
}

// Health Check
export interface HealthCheck {
  sys_status: string;         // "Healthy"
  uptime: number;             // Int64 (seconds)
}

export interface HealthCheckResponse extends Message<HealthCheck> {
  type: "health_check";
}

// API Responses
export interface ActionResponse {
  success: boolean;
  message: string;
}

// WebSocket Message Types
export type WebSocketMessageType = 
  | "ping" 
  | "pong" 
  | "agent_metrics" 
  | "agent_directory_snapshot"
  | "agent_list"
  | "health_check";

export interface WebSocketMessage {
  type: WebSocketMessageType;
  payload?: unknown;
}

// Tree structure for file browser
export interface FileTreeNode {
  name: string;
  path: string;
  size: number;
  modified: string;
  type: "file" | "directory";
  children?: FileTreeNode[];
  isExpanded?: boolean;
}

// WebSocket connection state
export type ConnectionStatus = "connecting" | "connected" | "disconnected" | "reconnecting";

// Agent with enhanced status
export interface AgentWithStatus extends AgentInfo {
  isOnline: boolean;
  metrics?: HostMetrics;
  lastMetricsUpdate?: number;
}

// Environment config
export interface EnvConfig {
  apiBaseUrl: string;
  wsUrl: string;
  wsRole: string;
  wsPingInterval: number;
  wsPongTimeout: number;
  wsReconnectDelay: number;
  wsMaxReconnectAttempts: number;
}
