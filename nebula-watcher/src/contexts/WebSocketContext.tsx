import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from "react";
import { wsService } from "@/services/websocket";
import type {
  ConnectionStatus,
  WebSocketMessage,
  MetricsPayload,
  DirectorySnapshot,
  AgentWithStatus,
  AgentInfo,
} from "@/types";

interface WebSocketContextValue {
  status: ConnectionStatus;
  agents: Map<string, AgentWithStatus>;
  metrics: Map<string, MetricsPayload>;
  snapshots: Map<string, DirectorySnapshot>;
  updateAgentList: (agents: AgentInfo[]) => void;
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null);

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const [agents, setAgents] = useState<Map<string, AgentWithStatus>>(new Map());
  const [metrics, setMetrics] = useState<Map<string, MetricsPayload>>(new Map());
  const [snapshots, setSnapshots] = useState<Map<string, DirectorySnapshot>>(new Map());

  const updateAgentList = useCallback((agentList: AgentInfo[]) => {
    console.log("[WebSocket] Updating agent list from API:", agentList);
    setAgents((prev) => {
      const newAgents = new Map(prev);
      const now = Date.now();
      const onlineThreshold = 90000; // 90 seconds (3 heartbeat cycles)

      agentList.forEach((agent) => {
        const lastSeen = new Date(agent.agent_last_seen).getTime();
        const existing = newAgents.get(agent.agent_id);
        
        // Check if online based on lastSeen AND last metrics update
        const metricsAge = existing?.lastMetricsUpdate 
          ? now - existing.lastMetricsUpdate 
          : Infinity;
        const isOnline = (now - lastSeen < onlineThreshold) && (metricsAge < onlineThreshold);
        
        console.log(`[WebSocket] Agent ${agent.agent_id} (${agent.agent_name || 'unnamed'}): lastSeen=${now - lastSeen}ms ago, metricsAge=${metricsAge}ms, isOnline=${isOnline}`);
        
        newAgents.set(agent.agent_id, {
          ...agent,
          isOnline,
          metrics: existing?.metrics,
          lastMetricsUpdate: existing?.lastMetricsUpdate,
        });
      });

      return newAgents;
    });
  }, []);

  useEffect(() => {
    // Connect on mount
    wsService.connect();

    // Subscribe to status changes
    const unsubStatus = wsService.onStatusChange(setStatus);

    // Subscribe to messages
    const unsubMessage = wsService.onMessage((message: WebSocketMessage) => {
      console.log("[WebSocket] Message received:", {
        type: message.type,
        payload: message.payload,
        timestamp: new Date().toISOString()
      });
      
      switch (message.type) {
        case "agent_metrics": {
          const payload = message.payload as MetricsPayload;
          console.log("[WebSocket] Processing agent_metrics:", {
            agent_id: payload.agent_id,
            metrics: payload.host_metrics,
            timestamp: payload.timestamp
          });
          
          // Always store metrics
          setMetrics((prev) => {
            const updated = new Map(prev);
            updated.set(payload.agent_id, payload);
            console.log(`[WebSocket] Stored metrics for ${payload.agent_id}, total metrics: ${updated.size}`);
            return updated;
          });
          
          // Update agent - create if doesn't exist
          setAgents((prev) => {
            const updated = new Map(prev);
            const existing = updated.get(payload.agent_id);
            const now = Date.now();
            
            if (existing) {
              // Update existing agent
              updated.set(payload.agent_id, {
                ...existing,
                isOnline: true,
                metrics: payload.host_metrics,
                lastMetricsUpdate: now,
                agent_last_seen: new Date().toISOString(),
              });
              console.log(`[WebSocket] Updated agent ${payload.agent_id} with metrics`);
            } else {
              // Create new agent entry from metrics
              updated.set(payload.agent_id, {
                agent_id: payload.agent_id,
                agent_name: undefined, // Will be updated from API
                agent_os: payload.host_metrics.os,
                agent_last_seen: new Date().toISOString(),
                isOnline: true,
                metrics: payload.host_metrics,
                lastMetricsUpdate: now,
              });
              console.log(`[WebSocket] Created new agent entry for ${payload.agent_id}`);
            }
            
            return updated;
          });
          break;
        }
        case "agent_directory_snapshot": {
          const payload = message.payload as DirectorySnapshot;
          console.log("[WebSocket] Processing directory snapshot:", {
            agent_id: payload.agent_id,
            fileCount: payload.directory?.total_files,
            totalSize: payload.directory?.total_size,
            timestamp: payload.timestamp
          });
          setSnapshots((prev) => {
            const updated = new Map(prev);
            updated.set(payload.agent_id, payload);
            console.log(`[WebSocket] Stored snapshot for ${payload.agent_id}, total snapshots: ${updated.size}`);
            return updated;
          });
          break;
        }
        default:
          console.log("[WebSocket] Unhandled message type:", message.type);
      }
    });

    return () => {
      unsubStatus();
      unsubMessage();
      wsService.disconnect();
    };
  }, []);

  return (
    <WebSocketContext.Provider
      value={{ status, agents, metrics, snapshots, updateAgentList }}
    >
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error("useWebSocket must be used within a WebSocketProvider");
  }
  return context;
}
