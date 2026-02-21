import { env } from "@/config/env";
import type { ConnectionStatus, WebSocketMessage } from "@/types";

type MessageHandler = (message: WebSocketMessage) => void;
type StatusHandler = (status: ConnectionStatus) => void;

class SSEService {
  private eventSource: EventSource | null = null;
  private messageHandlers: Set<MessageHandler> = new Set();
  private statusHandlers: Set<StatusHandler> = new Set();
  private reconnectAttempts = 0;
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private status: ConnectionStatus = "disconnected";
  private isIntentionalClose = false;

  connect(): void {
    if (this.eventSource?.readyState === EventSource.OPEN) {
      return;
    }

    this.isIntentionalClose = false;
    this.setStatus("connecting");

    // Convert WebSocket URL to SSE URL
    // If wsUrl is like "ws://localhost:8430/ws", convert to "http://localhost:8430/sse"
    let sseUrl = env.wsUrl.replace("ws://", "http://").replace("wss://", "https://");
    if (sseUrl.endsWith("/ws")) {
      sseUrl = sseUrl.replace("/ws", "/sse");
    } else {
      // If no /ws suffix, just append /sse
      sseUrl = sseUrl.endsWith("/") ? sseUrl + "sse" : sseUrl + "/sse";
    }
    
    try {
      this.eventSource = new EventSource(sseUrl);
      this.setupEventHandlers();
    } catch (error) {
      console.error("SSE connection error:", error);
      this.handleReconnect();
    }
  }

  private setupEventHandlers(): void {
    if (!this.eventSource) return;

    this.eventSource.onopen = () => {
      console.log("SSE connected");
      this.reconnectAttempts = 0;
      this.setStatus("connected");
    };

    this.eventSource.onerror = (error) => {
      console.error("SSE error:", error);
      
      if (this.eventSource?.readyState === EventSource.CLOSED) {
        // Connection closed, attempt to reconnect
        if (!this.isIntentionalClose) {
          this.setStatus("reconnecting");
          this.handleReconnect();
        } else {
          this.setStatus("disconnected");
        }
      }
    };

    this.eventSource.onmessage = (event) => {
      // Ignore keepalive comments (lines starting with :)
      if (event.data.trim().startsWith(":")) {
        return;
      }

      try {
        const message: WebSocketMessage = JSON.parse(event.data);
        console.log("SSE message received:", {
          type: message.type,
          payload: message.payload,
          timestamp: new Date().toISOString()
        });

        // Ignore connected message (just confirmation)
        if (message.type !== "connected") {
          this.notifyHandlers(message);
        }
      } catch (error) {
        console.error("Failed to parse SSE message:", error, event.data);
      }
    };
  }

  private handleReconnect(): void {
    if (this.reconnectAttempts >= 5) { // Max 5 attempts
      console.error("Max reconnect attempts reached");
      this.setStatus("disconnected");
      return;
    }

    this.setStatus("reconnecting");
    this.reconnectAttempts++;
    
    const delay = 1000 * Math.min(this.reconnectAttempts, 5); // Exponential backoff, max 5s
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

    this.reconnectTimeout = setTimeout(() => {
      this.connect();
    }, delay);
  }

  private cleanup(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
  }

  disconnect(): void {
    this.isIntentionalClose = true;
    this.cleanup();
    
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
    
    this.setStatus("disconnected");
  }

  private setStatus(status: ConnectionStatus): void {
    this.status = status;
    this.statusHandlers.forEach((handler) => handler(status));
  }

  getStatus(): ConnectionStatus {
    return this.status;
  }

  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  onStatusChange(handler: StatusHandler): () => void {
    this.statusHandlers.add(handler);
    // Immediately notify of current status
    handler(this.status);
    return () => this.statusHandlers.delete(handler);
  }

  private notifyHandlers(message: WebSocketMessage): void {
    this.messageHandlers.forEach((handler) => handler(message));
  }

  // SSE is one-way, so we don't need a send method
  // Frontend uses REST API for commands
}

export const sseService = new SSEService();

