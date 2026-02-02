import { useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  Server,
  Clock,
  Cpu,
  HardDrive,
  MemoryStick,
  Monitor,
  RefreshCw,
  Loader2,
} from "lucide-react";
import { Layout } from "@/components/layout/Layout";
import { MetricsGauge } from "@/components/agent/MetricsGauge";
import { FileTree } from "@/components/agent/FileTree";
import { AgentActions } from "@/components/agent/AgentActions";
import { Button } from "@/components/ui/button";
import { useAgent } from "@/hooks/useAgents";
import { api } from "@/services/api";
import { useToast } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";

export default function AgentDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const decodedId = id ? decodeURIComponent(id) : "";
  const { agent, metrics, snapshot, isLoading } = useAgent(decodedId);
  const { toast } = useToast();
  const [isRefreshingMetrics, setIsRefreshingMetrics] = useState(false);

  const handleRefreshMetrics = async () => {
    if (!decodedId) return;
    
    setIsRefreshingMetrics(true);
    try {
      await api.getAgentMetrics(decodedId);
      toast({
        title: "Metrics Requested",
        description: "Fresh metrics have been requested from the agent.",
      });
    } catch (error) {
      toast({
        title: "Request Failed",
        description: error instanceof Error ? error.message : "Failed to request metrics",
        variant: "destructive",
      });
    } finally {
      setIsRefreshingMetrics(false);
    }
  };

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    
    if (days > 0) return `${days}d ${hours}h ${mins}m`;
    if (hours > 0) return `${hours}h ${mins}m`;
    return `${mins}m`;
  };

  const osIcons: Record<string, string> = {
    Windows: "🪟",
    Linux: "🐧",
    darwin: "🍎",
  };

  if (isLoading) {
    return (
      <Layout>
        <div className="flex items-center justify-center h-screen">
          <Loader2 className="h-10 w-10 text-primary animate-spin" />
        </div>
      </Layout>
    );
  }

  if (!agent) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-screen gap-4">
          <Server className="h-16 w-16 text-muted-foreground" />
          <h2 className="text-xl font-semibold">Agent Not Found</h2>
          <p className="text-muted-foreground">The requested agent could not be found.</p>
          <Button asChild>
            <Link to="/">Back to Dashboard</Link>
          </Button>
        </div>
      </Layout>
    );
  }

  // Get metrics from either the metrics map or from agent object
  const hostMetrics = metrics?.host_metrics || agent?.metrics;

  return (
    <Layout>
      {/* Header */}
      <div className="border-b border-border bg-card/50 backdrop-blur-sm px-8 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Button
              variant="ghost"
              size="sm"
              asChild
              className="gap-2"
            >
              <Link to="/">
                <ArrowLeft className="h-4 w-4" />
                Back
              </Link>
            </Button>
            
            <div className="h-8 w-px bg-border" />
            
            <div className="flex items-center gap-3">
              <div className={cn(
                "h-3 w-3 rounded-full",
                agent.isOnline ? "bg-success glow-success" : "bg-destructive"
              )} />
              <div>
                <h1 className="text-xl font-semibold">{agent.agent_name || agent.agent_id}</h1>
                {agent.agent_name && (
                  <p className="text-sm font-mono text-muted-foreground">{agent.agent_id}</p>
                )}
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <span>{osIcons[agent.agent_os] || "💻"}</span>
                  <span>{agent.agent_os}</span>
                  {hostMetrics?.hostname && (
                    <>
                      <span>•</span>
                      <span>{hostMetrics.hostname}</span>
                    </>
                  )}
                </div>
              </div>
            </div>
          </div>

          <AgentActions
            agentId={decodedId}
            onActionComplete={() => navigate("/")}
          />
        </div>
      </div>

      {/* Content */}
      <div className="p-8 space-y-8 animate-fade-in">
        {/* Metrics Section */}
        <div className="glass-card rounded-xl p-6">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <Monitor className="h-5 w-5 text-primary" />
              <h2 className="text-lg font-semibold">System Metrics</h2>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={handleRefreshMetrics}
              disabled={isRefreshingMetrics}
              className="gap-2"
            >
              {isRefreshingMetrics ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4" />
              )}
              Refresh Metrics
            </Button>
          </div>

          {hostMetrics ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
              <MetricsGauge
                value={hostMetrics.cpu_usage}
                label="CPU Usage"
                icon={<Cpu className="h-5 w-5 text-muted-foreground" />}
                size="lg"
              />
              <MetricsGauge
                value={hostMetrics.memory_usage}
                label="Memory Usage"
                icon={<MemoryStick className="h-5 w-5 text-muted-foreground" />}
                size="lg"
              />
              <MetricsGauge
                value={hostMetrics.disk_usage}
                label="Disk Usage"
                icon={<HardDrive className="h-5 w-5 text-muted-foreground" />}
                size="lg"
              />
              
              {/* Uptime */}
              <div className="flex flex-col items-center justify-center gap-2">
                <div className="p-4 rounded-full bg-primary/10">
                  <Clock className="h-8 w-8 text-primary" />
                </div>
                <span className="text-2xl font-mono font-bold text-foreground">
                  {formatUptime(hostMetrics.uptime)}
                </span>
                <span className="text-sm text-muted-foreground">Uptime</span>
              </div>
            </div>
          ) : (
            <div className="text-center py-12">
              <Monitor className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <p className="text-muted-foreground">
                No metrics available yet. Click "Refresh Metrics" to request data.
              </p>
            </div>
          )}

          {/* Additional Info */}
          {hostMetrics && (
            <div className="mt-6 pt-6 border-t border-border">
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <InfoItem label="Hostname" value={hostMetrics.hostname} />
                <InfoItem label="Operating System" value={hostMetrics.os} />
                <InfoItem
                  label="Last Seen"
                  value={new Date(agent.agent_last_seen).toLocaleString()}
                />
                <InfoItem
                  label="Status"
                  value={agent.isOnline ? "Online" : "Offline"}
                  valueClassName={agent.isOnline ? "text-success" : "text-destructive"}
                />
              </div>
            </div>
          )}
        </div>

        {/* File Browser Section */}
        <FileTree snapshot={snapshot} />
      </div>
    </Layout>
  );
}

function InfoItem({
  label,
  value,
  valueClassName,
}: {
  label: string;
  value: string;
  valueClassName?: string;
}) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground uppercase tracking-wide">{label}</dt>
      <dd className={cn("text-sm font-medium mt-1", valueClassName)}>{value}</dd>
    </div>
  );
}
