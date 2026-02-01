import { useState } from "react";
import { RefreshCw, Trash2, AlertTriangle, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { api } from "@/services/api";
import { useToast } from "@/hooks/use-toast";

interface AgentActionsProps {
  agentId: string;
  onActionComplete?: () => void;
}

export function AgentActions({ agentId, onActionComplete }: AgentActionsProps) {
  const [isRestarting, setIsRestarting] = useState(false);
  const [isUninstalling, setIsUninstalling] = useState(false);
  const { toast } = useToast();

  const handleRestart = async () => {
    setIsRestarting(true);
    try {
      const result = await api.restartAgent(agentId);
      toast({
        title: "Restart Initiated",
        description: result.message || "Agent restart has been initiated.",
      });
      onActionComplete?.();
    } catch (error) {
      toast({
        title: "Restart Failed",
        description: error instanceof Error ? error.message : "Failed to restart agent",
        variant: "destructive",
      });
    } finally {
      setIsRestarting(false);
    }
  };

  const handleUninstall = async () => {
    setIsUninstalling(true);
    try {
      const result = await api.uninstallAgent(agentId);
      toast({
        title: "Uninstall Initiated",
        description: result.message || "Agent uninstallation has been initiated.",
      });
      onActionComplete?.();
    } catch (error) {
      toast({
        title: "Uninstall Failed",
        description: error instanceof Error ? error.message : "Failed to uninstall agent",
        variant: "destructive",
      });
    } finally {
      setIsUninstalling(false);
    }
  };

  return (
    <div className="flex items-center gap-3">
      <Button
        variant="outline"
        size="sm"
        onClick={handleRestart}
        disabled={isRestarting}
        className="gap-2"
      >
        {isRestarting ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <RefreshCw className="h-4 w-4" />
        )}
        Restart Agent
      </Button>

      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button
            variant="destructive"
            size="sm"
            disabled={isUninstalling}
            className="gap-2"
          >
            {isUninstalling ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Trash2 className="h-4 w-4" />
            )}
            Uninstall
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent className="glass-card border-destructive/30">
          <AlertDialogHeader>
            <AlertDialogTitle className="flex items-center gap-2 text-destructive">
              <AlertTriangle className="h-5 w-5" />
              Uninstall Agent
            </AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to uninstall <span className="font-mono text-foreground">{agentId}</span>?
              This action cannot be undone and will remove the agent from the remote system.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleUninstall}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Uninstall
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
