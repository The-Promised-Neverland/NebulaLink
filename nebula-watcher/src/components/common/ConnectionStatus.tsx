import { useWebSocket } from "@/contexts/WebSocketContext";

export function ConnectionStatus() {
  const { status } = useWebSocket();

  const statusConfig = {
    connected: {
      color: "bg-success",
      glow: "shadow-[0_0_10px_hsl(var(--success)/0.5)]",
      text: "Connected",
    },
    connecting: {
      color: "bg-warning",
      glow: "shadow-[0_0_10px_hsl(var(--warning)/0.5)]",
      text: "Connecting...",
    },
    disconnected: {
      color: "bg-destructive",
      glow: "shadow-[0_0_10px_hsl(var(--destructive)/0.5)]",
      text: "Disconnected",
    },
    reconnecting: {
      color: "bg-warning",
      glow: "shadow-[0_0_10px_hsl(var(--warning)/0.5)]",
      text: "Reconnecting...",
    },
  };

  const config = statusConfig[status];

  return (
    <div className="flex items-center gap-2">
      <div className="relative flex items-center">
        <div
          className={`h-2.5 w-2.5 rounded-full ${config.color} ${config.glow}`}
        />
        {status === "connected" && (
          <div
            className={`absolute inset-0 h-2.5 w-2.5 rounded-full ${config.color} animate-ping opacity-75`}
          />
        )}
      </div>
      <span className="text-sm text-muted-foreground">{config.text}</span>
    </div>
  );
}
