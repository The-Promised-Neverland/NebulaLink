import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from "react";
import { sseService } from "@/services/sse";
import type {
  ConnectionStatus,
  WebSocketMessage,
  MetricsPayload,
  DirectorySnapshot,
  AgentWithStatus,
  AgentInfo,
  TransferInfo,
  TransferStatusPayload,
} from "@/types";

interface WebSocketContextValue {
  status: ConnectionStatus;
  agents: Map<string, AgentWithStatus>;
  metrics: Map<string, MetricsPayload>;
  snapshots: Map<string, DirectorySnapshot>;
  transfers: Map<string, TransferInfo>; // Key: "sourceAgentId:path"
  updateAgentList: (agents: AgentInfo[]) => void;
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null);

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const [agents, setAgents] = useState<Map<string, AgentWithStatus>>(new Map());
  const [metrics, setMetrics] = useState<Map<string, MetricsPayload>>(new Map());
  const [snapshots, setSnapshots] = useState<Map<string, DirectorySnapshot>>(new Map());
  const [transfers, setTransfers] = useState<Map<string, TransferInfo>>(new Map());

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
          console.log("Processing directory snapshot for:", payload.agent_id, payload);
          setSnapshots((prev) => new Map(prev).set(payload.agent_id, payload));
          break;
        }
        case "master_filetransfer_manager": {
          const payload = message.payload as TransferStatusPayload;
          console.log("Processing transfer status:", payload);
          if (payload.status && payload.agent_id) {
            // Use agent_id as the key since we don't have path in the payload
            // The path will be tracked separately in FileTree component
            const transferKey = `${payload.source_agent_id || payload.agent_id}:${payload.agent_id}`;
            setTransfers((prev) => {
              const updated = new Map(prev);
              updated.set(transferKey, {
                status: payload.status,
                sourceAgentId: payload.source_agent_id || payload.agent_id,
                path: "", // Path not available in payload, will be tracked in component
                timestamp: Date.now(),
              });
              // Remove completed/failed transfers after 5 seconds
              if (payload.status === "completed" || payload.status === "failed") {
                setTimeout(() => {
                  setTransfers((prev) => {
                    const next = new Map(prev);
                    next.delete(transferKey);
                    return next;
                  });
                }, 5000);
              }
              return updated;
            });
          }
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
      value={{ status, agents, metrics, snapshots, transfers, updateAgentList }}
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
