import { Server, Activity, AlertTriangle, CheckCircle } from "lucide-react";
import type { AgentWithStatus } from "@/types";

interface SystemHealthProps {
  agents: AgentWithStatus[];
  isLoading?: boolean;
}

export function SystemHealth({ agents, isLoading }: SystemHealthProps) {
  const totalAgents = agents.length;
  const onlineAgents = agents.filter((a) => a.isOnline).length;
  const offlineAgents = totalAgents - onlineAgents;
  const healthPercentage = totalAgents > 0 ? (onlineAgents / totalAgents) * 100 : 0;

  const stats = [
    {
      label: "Total Agents",
      value: totalAgents,
      icon: Server,
      color: "text-primary",
      bgColor: "bg-primary/10",
    },
    {
      label: "Online",
      value: onlineAgents,
      icon: CheckCircle,
      color: "text-success",
      bgColor: "bg-success/10",
    },
    {
      label: "Offline",
      value: offlineAgents,
      icon: AlertTriangle,
      color: "text-destructive",
      bgColor: "bg-destructive/10",
    },
  ];

  return (
    <div className="glass-card p-6 rounded-xl">
      <div className="flex items-center gap-3 mb-6">
        <div className="p-2 rounded-lg bg-primary/10">
          <Activity className="h-5 w-5 text-primary" />
        </div>
        <div>
          <h2 className="text-lg font-semibold text-foreground">System Health</h2>
          <p className="text-sm text-muted-foreground">Real-time agent status</p>
        </div>
      </div>

      {/* Health bar */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-2">
          <span className="text-sm text-muted-foreground">Fleet Health</span>
          <span className="text-sm font-mono text-foreground">{healthPercentage.toFixed(0)}%</span>
        </div>
        <div className="h-2 rounded-full bg-muted overflow-hidden">
          <div
            className="h-full rounded-full bg-gradient-to-r from-success to-primary transition-all duration-500"
            style={{ width: `${healthPercentage}%` }}
          />
        </div>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-3 gap-4">
        {stats.map((stat) => (
          <div
            key={stat.label}
            className="text-center p-3 rounded-lg bg-muted/30"
          >
            <div className={`inline-flex p-2 rounded-lg ${stat.bgColor} mb-2`}>
              <stat.icon className={`h-4 w-4 ${stat.color}`} />
            </div>
            <div className={`text-2xl font-bold ${stat.color} font-mono`}>
              {isLoading ? "-" : stat.value}
            </div>
            <div className="text-xs text-muted-foreground mt-1">{stat.label}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
