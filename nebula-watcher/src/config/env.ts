import type { EnvConfig } from "@/types";

export const env: EnvConfig = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL || "http://localhost:8430",
  wsUrl: import.meta.env.VITE_WS_URL || "ws://localhost:8430/ws",
  wsRole: import.meta.env.VITE_WS_ROLE || "frontend",
  wsPingInterval: parseInt(import.meta.env.VITE_WS_PING_INTERVAL || "30000", 10),
  wsPongTimeout: parseInt(import.meta.env.VITE_WS_PONG_TIMEOUT || "60000", 10),
  wsReconnectDelay: parseInt(import.meta.env.VITE_WS_RECONNECT_DELAY || "3000", 10),
  wsMaxReconnectAttempts: parseInt(import.meta.env.VITE_WS_MAX_RECONNECT_ATTEMPTS || "10", 10),
};
