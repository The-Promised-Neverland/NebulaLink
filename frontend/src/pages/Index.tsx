import { useState, useEffect } from "react";
import { useWebSocket } from "@/hooks/useWebSocket";
import { useAgentWebSocket } from "@/hooks/useAgentWebSocket";
import { Agent } from "@/types/agent";
import { MetricCard } from "@/components/MetricCard";
import { AgentTable } from "@/components/AgentTable";
import { MetricsChart } from "@/components/MetricsChart";
import { EventLog } from "@/components/EventLog";
import { TaskControls } from "@/components/TaskControls";
import { StatusIndicator } from "@/components/StatusIndicator";
import { Server, Activity, HardDrive, Wifi } from "lucide-react";
import { useToast } from "@/hooks/use-toast";

const Index = () => {
  const { isConnected, registerHandler, eventLogs, addLog } = useWebSocket();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedAgentId, setSelectedAgentId] = useState<string>();
  const [metricsHistory, setMetricsHistory] = useState<Array<{
    timestamp: string;
    cpu: number;
    memory: number;
    disk: number;
  }>>([]);
  const { toast } = useToast();

  // Register WebSocket message handlers
  useAgentWebSocket({
    registerHandler,
    addLog,
    setAgents,
    setMetricsHistory,
    selectedAgentId,
  });

  // Check for offline agents every 15 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      const now = Date.now();
      setAgents(prev =>
        prev.map(agent => {
          const lastSeen = new Date(agent.agent_last_seen).getTime();
          const timeDiff = now - lastSeen;
          // Mark as offline if no heartbeat for 30 seconds
          if (timeDiff > 30000 && agent.status === 'online') {
            addLog(`Agent ${agent.agent_id} went offline`, 'warning');
            return { ...agent, status: 'offline' as const };
          }
          return agent;
        })
      );
    }, 15000);

    return () => clearInterval(interval);
  }, [addLog]);

  const onlineAgents = agents.filter(a => a.status === 'online').length;
  const offlineAgents = agents.filter(a => a.status === 'offline').length;

  const handleTriggerMetrics = async () => {
    // TODO: Implement HTTP API call
    addLog('Metrics trigger - API not configured yet', 'warning');
    toast({
      title: "API Not Configured",
      description: "Please configure API endpoints in src/config/api.ts",
      variant: "destructive",
    });
  };

  const handleAssignTask = async (agentId: string, taskName: string) => {
    // TODO: Implement HTTP API call
    addLog(`Task assignment - API not configured yet`, 'warning');
    toast({
      title: "API Not Configured",
      description: "Please configure API endpoints in src/config/api.ts",
      variant: "destructive",
    });
  };

  const handleRestartAgent = async (agentId: string) => {
    // TODO: Implement HTTP API call
    addLog(`Agent restart - API not configured yet`, 'warning');
    toast({
      title: "API Not Configured",
      description: "Please configure API endpoints in src/config/api.ts",
      variant: "destructive",
    });
  };

  const handleUninstallAgent = async (agentId: string) => {
    // TODO: Implement HTTP API call
    addLog(`Agent uninstall - API not configured yet`, 'warning');
    toast({
      title: "API Not Configured",
      description: "Please configure API endpoints in src/config/api.ts",
      variant: "destructive",
    });
  };

  return (
    <div className="relative min-h-screen bg-background p-6" style={{ position: 'relative', zIndex: 2 }}>
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center justify-between mb-2">
          <div className="relative">
            <h1 className="text-5xl font-display font-black text-foreground mb-2 text-glow-strong uppercase tracking-wider animate-float">
             🤖 NebulaLink
            </h1>
            <div className="h-1 w-32 bg-gradient-to-r from-primary via-secondary to-accent rounded-full mb-2" />
            <p className="text-muted-foreground font-body text-lg uppercase tracking-widest">Master Server Dashboard</p>
          </div>
          <div className="flex items-center gap-4">
            <div className="glass-effect px-6 py-3 rounded-lg border border-primary/30">
              <StatusIndicator
                status={isConnected ? 'online' : 'offline'}
                size="lg"
                showLabel
              />
            </div>
          </div>
        </div>
      </div>

      {/* Metrics Overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <MetricCard
          title="Total Agents"
          value={agents.length}
          icon={Server}
          trend="neutral"
        />
        <MetricCard
          title="Online Agents"
          value={onlineAgents}
          icon={Activity}
          trend="up"
        />
        <MetricCard
          title="Offline Agents"
          value={offlineAgents}
          icon={HardDrive}
          trend={offlineAgents > 0 ? 'down' : 'neutral'}
        />
        <MetricCard
          title="Connection"
          value={isConnected ? 'Active' : 'Disconnected'}
          icon={Wifi}
          trend={isConnected ? 'up' : 'down'}
        />
      </div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-8">
        {/* Agent Table - Takes 2 columns */}
        <div className="lg:col-span-2">
          <AgentTable
            agents={agents}
            onSelectAgent={setSelectedAgentId}
            selectedAgentId={selectedAgentId}
          />
        </div>

        {/* Task Controls - Takes 1 column */}
        <div>
          <TaskControls
            onTriggerMetrics={handleTriggerMetrics}
            onAssignTask={handleAssignTask}
            onRestartAgent={handleRestartAgent}
            onUninstallAgent={handleUninstallAgent}
            agents={agents}
          />
        </div>
      </div>

      {/* Bottom Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Metrics Chart */}
        <div>
          {selectedAgentId && metricsHistory.length > 0 ? (
            <MetricsChart data={metricsHistory} />
          ) : (
            <div className="relative glass-effect border-primary/20 rounded-lg p-8 flex flex-col items-center justify-center h-[400px] overflow-hidden group">
              {/* Animated grid background */}
              <div className="absolute inset-0 opacity-20">
                <div className="absolute inset-0 bg-gradient-to-br from-primary/10 via-transparent to-secondary/10" />
              </div>
              
              {/* Corner accents */}
              <div className="absolute top-0 left-0 w-20 h-20 border-t-2 border-l-2 border-primary/30 group-hover:border-primary/50 transition-colors" />
              <div className="absolute top-0 right-0 w-20 h-20 border-t-2 border-r-2 border-primary/30 group-hover:border-primary/50 transition-colors" />
              <div className="absolute bottom-0 left-0 w-20 h-20 border-b-2 border-l-2 border-primary/30 group-hover:border-primary/50 transition-colors" />
              <div className="absolute bottom-0 right-0 w-20 h-20 border-b-2 border-r-2 border-primary/30 group-hover:border-primary/50 transition-colors" />
              
              <div className="relative z-10 text-center animate-float">
                <Activity className="w-16 h-16 text-primary mx-auto mb-4 opacity-50" />
                <p className="text-muted-foreground font-display text-lg uppercase tracking-wider">
                  Select an agent to view metrics
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Event Log */}
        <div>
          <EventLog logs={eventLogs} />
        </div>
      </div>
    </div>
  );
};

export default Index;
