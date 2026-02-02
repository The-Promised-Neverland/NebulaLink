import { env } from "@/config/env";
import type { ConnectionStatus, WebSocketMessage } from "@/types";

type MessageHandler = (message: WebSocketMessage) => void;
type StatusHandler = (status: ConnectionStatus) => void;

class WebSocketService {
  private ws: WebSocket | null = null;
  private messageHandlers: Set<MessageHandler> = new Set();
  private statusHandlers: Set<StatusHandler> = new Set();
  private reconnectAttempts = 0;
  private pingInterval: ReturnType<typeof setInterval> | null = null;
  private pongTimeout: ReturnType<typeof setTimeout> | null = null;
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private status: ConnectionStatus = "disconnected";
  private isIntentionalClose = false;

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    this.isIntentionalClose = false;
    this.setStatus("connecting");

    const wsUrl = `${env.wsUrl}?role=${env.wsRole}`;
    
    try {
      this.ws = new WebSocket(wsUrl);
      this.setupEventHandlers();
    } catch (error) {
      console.error("WebSocket connection error:", error);
      this.handleReconnect();
    }
  }

  private setupEventHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      console.log("WebSocket connected");
      this.reconnectAttempts = 0;
      this.setStatus("connected");
      this.startPingInterval();
    };

    this.ws.onclose = (event) => {
      console.log("WebSocket closed:", event.code, event.reason);
      this.cleanup();
      
      if (!this.isIntentionalClose) {
        this.handleReconnect();
      } else {
        this.setStatus("disconnected");
      }
    };

    this.ws.onerror = (error) => {
      console.error("WebSocket error:", error);
    };

    this.ws.onmessage = (event) => {
      // Reset pong timeout on any message (connection is alive)
      this.handlePong();
      
      // Handle text messages (JSON)
      if (typeof event.data === "string") {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          console.log("WebSocket message received:", message);
          
          // Ignore pong messages (handled by native WebSocket)
          if (message.type !== "pong") {
            this.notifyHandlers(message);
          }
        } catch (error) {
          console.error("Failed to parse WebSocket message:", error, event.data);
        }
      }
      // Binary messages are likely ping/pong frames, ignore them
    };
  }

  private startPingInterval(): void {
    this.stopPingInterval();
    
    // Note: Browser WebSocket API doesn't support sending ping frames directly
    // The server will send pings, and browser will automatically respond with pong
    // We just need to monitor that we're receiving messages (which resets timeout)
    // So we don't actually need to send pings from client
    this.resetPongTimeout();
  }

  private stopPingInterval(): void {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  private sendPing(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      // Send native WebSocket ping frame (not JSON)
      // Note: Browser WebSocket API doesn't expose ping directly, so we'll skip custom ping
      // The server will send pings, and we'll respond to them
      // For now, just reset the pong timeout to give server time to ping us
      this.resetPongTimeout();
    }
  }

  private resetPongTimeout(): void {
    // Clear existing timeout
    if (this.pongTimeout) {
      clearTimeout(this.pongTimeout);
    }
    
    // Set new timeout - if server doesn't ping us, we'll reconnect
    // Master server sends ping every 30s and expects pong within 60s
    // So we should timeout if we don't receive any message (including ping) for ~90s
    // This gives us 3 missed ping cycles (30s * 3) before reconnecting
    const timeout = env.wsPingInterval * 3; // 90 seconds (3 ping cycles)
    this.pongTimeout = setTimeout(() => {
      console.warn("No server ping received - reconnecting...");
      if (this.ws) {
        this.ws.close();
      }
    }, timeout);
  }

  private handlePong(): void {
    // Server sent pong (or we received any message, which means connection is alive)
    this.resetPongTimeout();
  }

  private handleReconnect(): void {
    if (this.reconnectAttempts >= env.wsMaxReconnectAttempts) {
      console.error("Max reconnect attempts reached");
      this.setStatus("disconnected");
      return;
    }

    this.setStatus("reconnecting");
    this.reconnectAttempts++;
    
    const delay = env.wsReconnectDelay * Math.min(this.reconnectAttempts, 5);
    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

    this.reconnectTimeout = setTimeout(() => {
      this.connect();
    }, delay);
  }

  private cleanup(): void {
    this.stopPingInterval();
    
    if (this.pongTimeout) {
      clearTimeout(this.pongTimeout);
      this.pongTimeout = null;
    }
    
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
  }

  disconnect(): void {
    this.isIntentionalClose = true;
    this.cleanup();
    
    if (this.ws) {
      this.ws.close();
      this.ws = null;
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

  send(message: WebSocketMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    } else {
      console.warn("WebSocket not connected, cannot send message");
    }
  }
}

export const wsService = new WebSocketService();
