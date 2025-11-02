import { useEffect } from 'react';
import { Agent, AgentMetrics } from '@/types/agent';
import { useToast } from '@/hooks/use-toast';

interface UseAgentWebSocketProps {
  registerHandler: (id: string, handler: (data: any) => void) => () => void;
  addLog: (message: string, type: 'info' | 'success' | 'error' | 'warning') => void;
  setAgents: React.Dispatch<React.SetStateAction<Agent[]>>;
  setMetricsHistory: React.Dispatch<React.SetStateAction<Array<{
    timestamp: string;
    cpu: number;
    memory: number;
    disk: number;
  }>>>;
  selectedAgentId?: string;
}

export const useAgentWebSocket = ({
  registerHandler,
  addLog,
  setAgents,
  setMetricsHistory,
  selectedAgentId,
}: UseAgentWebSocketProps) => {
  const { toast } = useToast();

  useEffect(() => {
    const unregister = registerHandler('main', (message) => {
      switch (message.type) {
        case 'agents_list':
          if (message.payload) {
            const agentsList: Agent[] = message.payload.map((agent: any) => ({
              agent_id: agent?.agent_id,
              agent_os: agent.agent_os,
              agent_last_seen: agent.agent_last_seen,
              status: 'online',
              uptime: agent.uptime || 0,
              metrics: agent.metrics,
            }));
            setAgents(agentsList);
            addLog(`Received ${agentsList.length} agents`, 'success');
          }
          break;

        case 'agent_connected':
          if (message.payload) {
            addLog(`Agent ${message.payload?.agent_id} connected`, 'success');
          }
          break;

        case 'agent_disconnected':
          if (message.payload) {
            const agentId = message.payload?.agent_id;
            setAgents(prev =>
              prev.map(agent =>
                agent?.agent_id === agentId
                  ? { ...agent, status: 'offline' as const }
                  : agent
              )
            );
            addLog(`Agent ${agentId} disconnected`, 'warning');
          }
          break;

        case 'metrics_update':
          if (message.payload) {
            const metrics: AgentMetrics = message.payload;
            setAgents(prev =>
              prev.map(agent =>
                agent?.agent_id === metrics?.agent_id
                  ? { ...agent, metrics, uptime: metrics.uptime }
                  : agent
              )
            );

            // Add to metrics history for charts
            if (selectedAgentId === metrics?.agent_id) {
              setMetricsHistory(prev => [
                ...prev.slice(-19),
                {
                  timestamp: new Date().toLocaleTimeString(),
                  cpu: metrics.cpu_usage,
                  memory: metrics.memory_usage,
                  disk: metrics.disk_usage,
                },
              ]);
            }
            addLog(`Metrics updated for ${metrics?.agent_id}`, 'info');
          }
          break;

        case 'agent_metrics':
          if (message.payload?.host_metrics) {
            const agentId = message.payload.agent_id;
            const hostMetrics = message.payload.host_metrics;
            const now = new Date().toISOString();

            setAgents(prev => {
              const existingAgent = prev.find(agent => agent?.agent_id === agentId);

              if (existingAgent) {
                // Update existing agent
                return prev.map(agent =>
                  agent?.agent_id === agentId
                    ? {
                      ...agent,
                      status: 'online' as const,
                      agent_last_seen: now,
                      agent_os: hostMetrics.os,
                      metrics: {
                        agent_id: agentId,
                        cpu_usage: hostMetrics.cpu_usage,
                        memory_usage: hostMetrics.memory_usage,
                        disk_usage: hostMetrics.disk_usage,
                        hostname: hostMetrics.hostname,
                        os: hostMetrics.os,
                        uptime: hostMetrics.uptime,
                      },
                      uptime: hostMetrics.uptime,
                    }
                    : agent
                );
              } else {
                // Add new agent
                addLog(`New agent connected: ${agentId}`, 'success');
                return [
                  ...prev,
                  {
                    agent_id: agentId,
                    agent_os: hostMetrics.os,
                    agent_last_seen: now,
                    status: 'online' as const,
                    uptime: hostMetrics.uptime,
                    metrics: {
                      agent_id: agentId,
                      cpu_usage: hostMetrics.cpu_usage,
                      memory_usage: hostMetrics.memory_usage,
                      disk_usage: hostMetrics.disk_usage,
                      hostname: hostMetrics.hostname,
                      os: hostMetrics.os,
                      uptime: hostMetrics.uptime,
                    },
                  },
                ];
              }
            });

            // Add to metrics history for charts
            if (selectedAgentId === agentId) {
              setMetricsHistory(prev => [
                ...prev.slice(-19),
                {
                  timestamp: new Date().toLocaleTimeString(),
                  cpu: hostMetrics.cpu_usage,
                  memory: hostMetrics.memory_usage,
                  disk: hostMetrics.disk_usage,
                },
              ]);
            }
          }
          break;

        case 'task_status':
          if (message.payload) {
            addLog(`Task ${message.payload.task_name} on ${message.payload?.agent_id}: ${message.payload.status}`, 'success');
            toast({
              title: "Task Status",
              description: `${message.payload.task_name}: ${message.payload.status}`,
            });
          }
          break;

        default:
          console.log('Unhandled message type:', message.type);
      }
    });

    return unregister;
  }, [registerHandler, addLog, selectedAgentId, toast, setAgents, setMetricsHistory]);
};
