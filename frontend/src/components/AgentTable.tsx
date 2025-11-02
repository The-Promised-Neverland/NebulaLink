import { Agent } from "@/types/agent";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { StatusIndicator } from "./StatusIndicator";
import { Card } from "@/components/ui/card";
import { formatDistanceToNow } from "date-fns";
import { Button } from "@/components/ui/button";
import { Activity } from "lucide-react";

interface AgentTableProps {
  agents: Agent[];
  onSelectAgent: (agentId: string) => void;
  selectedAgentId?: string;
}

const formatUptime = (seconds: number) => {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;
  return `${hours}h ${minutes}m ${secs}s`;
};

export const AgentTable = ({ agents, onSelectAgent, selectedAgentId }: AgentTableProps) => {
  return (
    <Card className="relative glass-effect border-primary/20 overflow-hidden">
      {/* Scanning line effect */}
      <div className="absolute inset-x-0 h-px bg-gradient-to-r from-transparent via-primary to-transparent opacity-50 animate-scan" />
      
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow className="border-primary/20 hover:bg-transparent">
              <TableHead className="text-muted-foreground font-display text-xs uppercase tracking-wider">Status</TableHead>
              <TableHead className="text-muted-foreground font-display text-xs uppercase tracking-wider">Agent ID</TableHead>
              <TableHead className="text-muted-foreground font-display text-xs uppercase tracking-wider">OS</TableHead>
              <TableHead className="text-muted-foreground font-display text-xs uppercase tracking-wider">Uptime</TableHead>
              <TableHead className="text-muted-foreground font-display text-xs uppercase tracking-wider">Last Seen</TableHead>
              <TableHead className="text-muted-foreground font-display text-xs uppercase tracking-wider">CPU</TableHead>
              <TableHead className="text-muted-foreground font-display text-xs uppercase tracking-wider">Memory</TableHead>
              <TableHead className="text-muted-foreground font-display text-xs uppercase tracking-wider">Disk</TableHead>
              <TableHead className="text-muted-foreground font-display text-xs uppercase tracking-wider">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {agents.length === 0 ? (
              <TableRow>
                <TableCell colSpan={9} className="text-center text-muted-foreground py-8">
                  No agents connected
                </TableCell>
              </TableRow>
            ) : (
              agents.map((agent) => (
                <TableRow
                  key={agent.agent_id}
                  className={`border-primary/10 hover:bg-primary/5 hover:border-glow transition-all duration-300 cursor-pointer ${
                    selectedAgentId === agent?.agent_id ? 'bg-primary/10 border-glow' : ''
                  }`}
                  onClick={() => onSelectAgent(agent?.agent_id)}
                >
                  <TableCell>
                    <StatusIndicator status={agent.status} size="sm" />
                  </TableCell>
                  <TableCell className="font-mono text-primary-bright font-semibold">{agent?.agent_id}</TableCell>
                  <TableCell className="text-foreground font-medium">{agent.agent_os}</TableCell>
                  <TableCell className="font-mono text-foreground tabular-nums">
                    {formatUptime(agent.uptime)}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {formatDistanceToNow(new Date(agent.agent_last_seen), { addSuffix: true })}
                  </TableCell>
                  <TableCell className="font-mono text-chart-1 font-semibold tabular-nums">
                    {agent.metrics?.cpu_usage?.toFixed(1) || '—'}%
                  </TableCell>
                  <TableCell className="font-mono text-chart-2 font-semibold tabular-nums">
                    {agent.metrics?.memory_usage?.toFixed(1) || '—'}%
                  </TableCell>
                  <TableCell className="font-mono text-chart-3 font-semibold tabular-nums">
                    {agent.metrics?.disk_usage?.toFixed(1) || '—'}%
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-primary hover:text-primary hover:bg-primary/10"
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelectAgent(agent?.agent_id);
                      }}
                    >
                      <Activity className="w-4 h-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </Card>
  );
};
