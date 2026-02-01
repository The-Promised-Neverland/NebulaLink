import { Layout } from "@/components/layout/Layout";
import { Header } from "@/components/layout/Header";
import { AgentCard } from "@/components/dashboard/AgentCard";
import { SystemHealth } from "@/components/dashboard/SystemHealth";
import { useAgents } from "@/hooks/useAgents";
import { Server, Loader2 } from "lucide-react";

export default function Dashboard() {
  const { agents, isLoading, refetch } = useAgents();

  return (
    <Layout>
      <Header
        title="Dashboard"
        subtitle="Monitor your distributed agent network"
        onRefresh={refetch}
      />

      <div className="p-8 animate-fade-in">
        {/* System Health Overview */}
        <div className="mb-8">
          <SystemHealth agents={agents} isLoading={isLoading} />
        </div>

        {/* Agent List */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Server className="h-5 w-5 text-primary" />
              <h2 className="text-lg font-semibold text-foreground">Agents</h2>
            </div>
            <span className="text-sm text-muted-foreground">
              {agents.length} registered
            </span>
          </div>

          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="h-8 w-8 text-primary animate-spin" />
            </div>
          ) : agents.length === 0 ? (
            <div className="glass-card rounded-xl p-12 text-center">
              <Server className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <h3 className="text-lg font-medium text-foreground mb-2">No Agents Found</h3>
              <p className="text-sm text-muted-foreground">
                No agents have connected to the master server yet.
                <br />
                Deploy agents to start monitoring.
              </p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {agents.map((agent) => (
                <AgentCard key={agent.agent_id} agent={agent} />
              ))}
            </div>
          )}
        </div>
      </div>
    </Layout>
  );
}
