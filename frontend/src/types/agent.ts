export interface Agent {
  agent_id?: string;
  agent_os?: string;
  agent_last_seen?: string;
  status?: 'online' | 'offline';
  uptime?: number;
  metrics?: AgentMetrics;
}

export interface AgentMetrics {
  agent_id?: string;
  cpu_usage?: number;
  memory_usage?: number;
  disk_usage?: number;
  hostname?: string;
  os?: string;
  uptime?: number;
}

export interface WSMessage {
  type?: string;
  payload?: any;
}

export interface EventLog {
  id?: string;
  timestamp?: Date;
  message?: string;
  type?: 'info' | 'success' | 'error' | 'warning';
}
