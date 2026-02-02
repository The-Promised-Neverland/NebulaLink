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
          
          // Extract agent ID - might be in format "agent:ID" or just "ID"
          let agentId = payload.agent_id;
          if (agentId.startsWith("agent:")) {
            agentId = agentId.substring(6); // Remove "agent:" prefix
          }
          
          console.log("[WebSocket] Processing agent_metrics:", {
            original_agent_id: payload.agent_id,
            normalized_agent_id: agentId,
            metrics: payload.host_metrics,
            timestamp: payload.timestamp
          });
          
          // Always store metrics with normalized ID
          setMetrics((prev) => {
            const updated = new Map(prev);
            updated.set(agentId, payload);
            console.log(`[WebSocket] Stored metrics for ${agentId}, total metrics: ${updated.size}`);
            return updated;
          });
          
          // Update agent - create if doesn't exist
          setAgents((prev) => {
            const updated = new Map(prev);
            const existing = updated.get(agentId);
            const now = Date.now();
            
            if (existing) {
              // Update existing agent
              updated.set(agentId, {
                ...existing,
                isOnline: true,
                metrics: payload.host_metrics,
                lastMetricsUpdate: now,
                agent_last_seen: new Date().toISOString(),
              });
              console.log(`[WebSocket] Updated agent ${agentId} with metrics`);
            } else {
              // Create new agent entry from metrics
              updated.set(agentId, {
                agent_id: agentId,
                agent_name: undefined, // Will be updated from API
                agent_os: payload.host_metrics.os,
                agent_last_seen: new Date().toISOString(),
                isOnline: true,
                metrics: payload.host_metrics,
                lastMetricsUpdate: now,
              });
              console.log(`[WebSocket] Created new agent entry for ${agentId}`);
            }
            
            return updated;
          });
          break;
        }
        case "agent_directory_snapshot": {
          const payload = message.payload as DirectorySnapshot;
          
          // Extract agent ID - might be in format "agent:ID" or just "ID"
          let agentId = payload.agent_id;
          if (agentId.startsWith("agent:")) {
            agentId = agentId.substring(6); // Remove "agent:" prefix
          }
          
          console.log("[WebSocket] Processing directory snapshot:", {
            original_agent_id: payload.agent_id,
            normalized_agent_id: agentId,
            fileCount: payload.directory?.total_files,
            totalSize: payload.directory?.total_size,
            timestamp: payload.timestamp
          });
          setSnapshots((prev) => {
            const updated = new Map(prev);
            updated.set(agentId, payload);
            console.log(`[WebSocket] Stored snapshot for ${agentId}, total snapshots: ${updated.size}`);
            return updated;
          });
          break;
        }
        case "agent_list": {
          const agentList = message.payload as AgentInfo[];
          console.log("[WebSocket] Received agent_list update:", agentList);
          // Update agent list directly from WebSocket (real-time update)
          updateAgentList(agentList);
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
