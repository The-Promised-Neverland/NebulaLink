import { Link } from "react-router-dom";
import { Server, Monitor, Clock, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import type { AgentWithStatus } from "@/types";

interface AgentCardProps {
  agent: AgentWithStatus;
}

export function AgentCard({ agent }: AgentCardProps) {
  const osIcons: Record<string, string> = {
    Windows: "🪟",
    Linux: "🐧",
    darwin: "🍎",
  };

  const formatLastSeen = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}d ago`;
    if (hours > 0) return `${hours}h ago`;
    if (minutes > 0) return `${minutes}m ago`;
    return "Just now";
  };

  return (
    <Link to={`/agent/${encodeURIComponent(agent.agent_id)}`}>
      <div
        className={cn(
          "glass-card p-5 rounded-xl transition-all duration-300 hover:scale-[1.02] group cursor-pointer",
          agent.isOnline
            ? "hover:shadow-[0_0_30px_hsl(var(--success)/0.2)]"
            : "hover:shadow-[0_0_30px_hsl(var(--destructive)/0.2)] opacity-75"
        )}
      >
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            {/* Status indicator */}
            <div className="relative">
              <div
                className={cn(
                  "h-3 w-3 rounded-full",
                  agent.isOnline ? "bg-success" : "bg-destructive"
                )}
              />
              {agent.isOnline && (
                <div className="absolute inset-0 h-3 w-3 rounded-full bg-success animate-ping opacity-50" />
              )}
            </div>
            
            <div>
              <h3 className="text-sm font-medium text-foreground group-hover:text-primary transition-colors">
                {agent.agent_name || agent.agent_id}
              </h3>
              <div className="flex items-center gap-2 mt-1">
                <span className="text-lg">{osIcons[agent.agent_os] || "💻"}</span>
                <span className="text-sm text-muted-foreground">{agent.agent_os}</span>
              </div>
            </div>
          </div>

          <ChevronRight className="h-5 w-5 text-muted-foreground group-hover:text-primary group-hover:translate-x-1 transition-all" />
        </div>

        {/* Metrics preview if available */}
        {agent.metrics && (
          <div className="mt-4 grid grid-cols-3 gap-2">
            <MetricPill label="CPU" value={agent.metrics.cpu_usage} />
            <MetricPill label="RAM" value={agent.metrics.memory_usage} />
            <MetricPill label="Disk" value={agent.metrics.disk_usage} />
          </div>
        )}

        {/* Last seen */}
        <div className="mt-4 flex items-center gap-2 text-xs text-muted-foreground">
          <Clock className="h-3.5 w-3.5" />
          <span>Last seen: {formatLastSeen(agent.agent_last_seen)}</span>
        </div>
      </div>
    </Link>
  );
}

function MetricPill({ label, value }: { label: string; value: number }) {
  const getColor = (val: number) => {
    if (val >= 90) return "text-destructive bg-destructive/10";
    if (val >= 70) return "text-warning bg-warning/10";
    return "text-success bg-success/10";
  };

  return (
    <div className={cn("px-2 py-1 rounded text-xs font-mono", getColor(value))}>
      {label}: {value.toFixed(0)}%
    </div>
  );
}
