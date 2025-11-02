import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Send, Activity, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";

interface TaskControlsProps {
  onTriggerMetrics: () => void;
  onAssignTask: (agentId: string, taskName: string) => void;
  onRestartAgent: (agentId: string) => void;
  onUninstallAgent: (agentId: string) => void;
  agents: Array<{ agent_id: string; status: string }>;
}

export const TaskControls = ({
  onTriggerMetrics,
  onAssignTask,
  onRestartAgent,
  onUninstallAgent,
  agents,
}: TaskControlsProps) => {
  const [selectedAgent, setSelectedAgent] = useState<string>("");
  const [taskName, setTaskName] = useState<string>("");

  const handleAssignTask = () => {
    if (selectedAgent && taskName) {
      onAssignTask(selectedAgent, taskName);
      setTaskName("");
    }
  };

  const handleRestart = () => {
    if (selectedAgent) {
      onRestartAgent(selectedAgent);
    }
  };

  const handleUninstall = () => {
    if (selectedAgent) {
      onUninstallAgent(selectedAgent);
    }
  };

  return (
    <Card className="relative p-6 glass-effect border-primary/20 overflow-hidden group">
      {/* Corner accents */}
      <div className="absolute top-0 left-0 w-12 h-12 border-t-2 border-l-2 border-primary/40 group-hover:border-primary/70 transition-colors" />
      <div className="absolute bottom-0 right-0 w-12 h-12 border-b-2 border-r-2 border-primary/40 group-hover:border-primary/70 transition-colors" />
      
      <h3 className="relative text-xl font-display font-bold mb-6 text-foreground text-glow uppercase tracking-wider">Task Controls</h3>
      <div className="space-y-4">
        <div>
          <Button
            onClick={onTriggerMetrics}
            className="w-full bg-primary hover:bg-primary-bright text-primary-foreground font-display font-bold hover:card-glow-strong transition-all duration-300 hover:scale-[1.02]"
          >
            <Activity className="w-4 h-4 mr-2" />
            Trigger Metrics (All Agents)
          </Button>
        </div>

        <div className="space-y-2">
          <Label htmlFor="agent-select" className="text-foreground font-display text-xs uppercase tracking-wider">Select Agent</Label>
          <Select value={selectedAgent} onValueChange={setSelectedAgent}>
            <SelectTrigger id="agent-select" className="bg-input border-primary/30 text-foreground hover:border-primary/60 transition-colors font-mono">
              <SelectValue placeholder="Choose an agent" />
            </SelectTrigger>
            <SelectContent className="bg-popover border-primary/30">
  {agents.length === 0 ? (
    <SelectItem disabled value="none">No agents available</SelectItem>
  ) : (
    agents
      .filter(agent => agent.agent_id && agent.agent_id.trim() !== "") // ✅ filter out blanks
      .map(agent => (
        <SelectItem
          key={agent.agent_id}
          value={agent.agent_id}
          className="text-foreground hover:bg-primary/10 font-mono"
        >
          {agent.agent_id} ({agent.status})
        </SelectItem>
      ))
  )}
</SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="task-name" className="text-foreground font-display text-xs uppercase tracking-wider">Task Name</Label>
          <Input
            id="task-name"
            placeholder="e.g., CleanupTemp"
            value={taskName}
            onChange={(e) => setTaskName(e.target.value)}
            className="bg-input border-primary/30 text-foreground placeholder:text-muted-foreground font-mono hover:border-primary/60 focus:border-primary transition-colors"
          />
        </div>

        <Button
          onClick={handleAssignTask}
          disabled={!selectedAgent || !taskName}
          className="w-full bg-secondary hover:bg-secondary/80 text-secondary-foreground font-display font-bold hover:card-glow transition-all duration-300 hover:scale-[1.02] disabled:opacity-50 disabled:hover:scale-100"
        >
          <Send className="w-4 h-4 mr-2" />
          Assign Task
        </Button>

        <div className="pt-4 border-t border-primary/20 space-y-2">
          <Button
            onClick={handleRestart}
            disabled={!selectedAgent}
            variant="outline"
            className="w-full border-warning/50 text-warning hover:bg-warning/10 hover:border-warning font-display font-bold transition-all duration-300 hover:scale-[1.02] disabled:opacity-50 disabled:hover:scale-100"
          >
            <RefreshCw className="w-4 h-4 mr-2" />
            Restart Agent
          </Button>

          <Button
            onClick={handleUninstall}
            disabled={!selectedAgent}
            variant="outline"
            className="w-full border-destructive/50 text-destructive hover:bg-destructive/10 hover:border-destructive font-display font-bold transition-all duration-300 hover:scale-[1.02] disabled:opacity-50 disabled:hover:scale-100"
          >
            <Trash2 className="w-4 h-4 mr-2" />
            Uninstall Agent
          </Button>
        </div>
      </div>
    </Card>
  );
};
