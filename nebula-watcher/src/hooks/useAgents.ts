import { useQuery } from "@tanstack/react-query";
import { api } from "@/services/api";
import { useWebSocket } from "@/contexts/WebSocketContext";
import { useEffect } from "react";
import type { AgentWithStatus, AgentInfo } from "@/types";

export function useAgents() {
  const { agents, updateAgentList } = useWebSocket();

  const query = useQuery({
    queryKey: ["agents"],
    queryFn: async () => {
      const response = await api.getAgents();
      return response.payload || [];
    },
    refetchInterval: 30000, // Refetch every 30 seconds
  });

  // Update WebSocket context when API data changes
  useEffect(() => {
    if (query.data) {
      updateAgentList(query.data);
    }
  }, [query.data, updateAgentList]);

  // Convert Map to array for easier consumption
  const agentList: AgentWithStatus[] = Array.from(agents.values());

  return {
    agents: agentList,
    isLoading: query.isLoading,
    error: query.error,
    refetch: query.refetch,
  };
}

export function useAgent(id: string): {
  agent: AgentWithStatus | undefined;
  metrics: ReturnType<typeof useWebSocket>["metrics"] extends Map<string, infer T> ? T | undefined : never;
  snapshot: ReturnType<typeof useWebSocket>["snapshots"] extends Map<string, infer T> ? T | undefined : never;
  isLoading: boolean;
  error: Error | null;
} {
  const { agents, metrics, snapshots } = useWebSocket();
  
  const agentQuery = useQuery({
    queryKey: ["agent", id],
    queryFn: async () => {
      const response = await api.getAgent(id);
      return response.payload;
    },
    enabled: !!id,
  });

  const agentFromContext = agents.get(id);
  const agentMetrics = metrics.get(id);
  const snapshot = snapshots.get(id);

  // Merge API data with context data, preferring context (which has isOnline status)
  const agent: AgentWithStatus | undefined = agentFromContext || (agentQuery.data ? {
    ...agentQuery.data,
    isOnline: false,
    metrics: undefined,
    lastMetricsUpdate: undefined,
  } : undefined);

  return {
    agent,
    metrics: agentMetrics,
    snapshot,
    isLoading: agentQuery.isLoading,
    error: agentQuery.error,
  };
}

export function useHealthCheck() {
  return useQuery({
    queryKey: ["health"],
    queryFn: async () => {
      const response = await api.getHealth();
      return response.payload;
    },
    refetchInterval: 10000, // Refetch every 10 seconds
  });
}
