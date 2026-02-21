import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from "react";
import { sseService } from "@/services/sse";
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
    setAgents((prev) => {
      const newAgents = new Map(prev);
      const now = Date.now();
      const onlineThreshold = 60000; // 1 minute

      agentList.forEach((agent) => {
        const lastSeen = new Date(agent.agent_last_seen).getTime();
        const isOnline = now - lastSeen < onlineThreshold;
        
        const existing = newAgents.get(agent.agent_id);
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
    sseService.connect();

    // Subscribe to status changes
    const unsubStatus = sseService.onStatusChange(setStatus);

    // Subscribe to messages
    const unsubMessage = sseService.onMessage((message: WebSocketMessage) => {
      console.log("WebSocket message received:", message.type, message.payload);
      
      switch (message.type) {
        case "agent_metrics": {
          const payload = message.payload as MetricsPayload;
          console.log("Processing agent_metrics for:", payload.agent_id, payload);
          
          // Always store metrics
          setMetrics((prev) => new Map(prev).set(payload.agent_id, payload));
          
          // Update agent - create if doesn't exist
          setAgents((prev) => {
            const updated = new Map(prev);
            const existing = updated.get(payload.agent_id);
            
            if (existing) {
              // Update existing agent
              updated.set(payload.agent_id, {
                ...existing,
                isOnline: true,
                metrics: payload.host_metrics,
                lastMetricsUpdate: Date.now(),
              });
            } else {
              // Create new agent entry from metrics
              updated.set(payload.agent_id, {
                agent_id: payload.agent_id,
                agent_os: payload.host_metrics.os,
                agent_last_seen: new Date().toISOString(),
                isOnline: true,
                metrics: payload.host_metrics,
                lastMetricsUpdate: Date.now(),
              });
            }
            
            return updated;
          });
          break;
        }
        case "agent_directory_snapshot": {
          const payload = message.payload as DirectorySnapshot;
          console.log("Processing directory snapshot for:", payload.agent_id);
          setSnapshots((prev) => new Map(prev).set(payload.agent_id, payload));
          break;
        }
      }
    });

    return () => {
      unsubStatus();
      unsubMessage();
      sseService.disconnect();
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
