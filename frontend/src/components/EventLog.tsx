import { Card } from "@/components/ui/card";
import { EventLog as EventLogType } from "@/types/agent";
import { format } from "date-fns";
import { cn } from "@/lib/utils";
import { AlertCircle, CheckCircle, Info, AlertTriangle } from "lucide-react";

interface EventLogProps {
  logs: EventLogType[];
}

export const EventLog = ({ logs }: EventLogProps) => {
  const getIcon = (type: EventLogType['type']) => {
    switch (type) {
      case 'success':
        return <CheckCircle className="w-4 h-4 text-success" />;
      case 'error':
        return <AlertCircle className="w-4 h-4 text-destructive" />;
      case 'warning':
        return <AlertTriangle className="w-4 h-4 text-warning" />;
      default:
        return <Info className="w-4 h-4 text-primary" />;
    }
  };

  const getTextColor = (type: EventLogType['type']) => {
    switch (type) {
      case 'success':
        return 'text-success';
      case 'error':
        return 'text-destructive';
      case 'warning':
        return 'text-warning';
      default:
        return 'text-foreground';
    }
  };

  return (
    <Card className="relative glass-effect border-primary/20 h-full overflow-hidden">
      {/* Scanning line */}
      <div className="absolute inset-x-0 h-px bg-gradient-to-r from-transparent via-primary to-transparent opacity-50 animate-scan" />
      
      <div className="p-4 border-b border-primary/20 bg-primary/5">
        <h3 className="text-xl font-display font-bold text-foreground text-glow uppercase tracking-wider">Event Log</h3>
      </div>
      <div className="h-[400px] overflow-y-auto p-4 space-y-2 font-mono text-sm custom-scrollbar">
        {logs.length === 0 ? (
          <div className="text-muted-foreground text-center py-8 animate-pulse-slow">
            <Info className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p className="font-display uppercase tracking-wider">No events yet</p>
          </div>
        ) : (
          logs.map((log) => (
            <div
              key={log.id}
              className="flex items-start gap-3 p-3 rounded-lg hover:bg-primary/5 hover:border-l-2 hover:border-primary transition-all duration-300 animate-fade-in group"
            >
              <div className="mt-0.5">{getIcon(log.type)}</div>
              <div className="flex-1 min-w-0">
                <span className="text-primary-bright text-xs font-semibold tabular-nums">
                  [{format(log.timestamp, 'HH:mm:ss')}]
                </span>
                <span className={cn('ml-2 group-hover:text-glow transition-all', getTextColor(log.type))}>
                  {log.message}
                </span>
              </div>
            </div>
          ))
        )}
      </div>
    </Card>
  );
};
