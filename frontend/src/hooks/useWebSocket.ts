import { useEffect, useRef, useState, useCallback } from 'react';
import { WSMessage, EventLog } from '@/types/agent';

const WS_URL = import.meta.env.VITE_WS_URL;

export const useWebSocket = () => {
  const [isConnected, setIsConnected] = useState(false);
  const [eventLogs, setEventLogs] = useState<EventLog[]>([]);
  const ws = useRef<WebSocket | null>(null);
  const reconnectTimeout = useRef<NodeJS.Timeout>();
  const messageHandlers = useRef<Map<string, (data: any) => void>>(new Map());

  const addLog = useCallback((message: string, type: EventLog['type'] = 'info') => {
    const log: EventLog = {
      id: Math.random().toString(36).substr(2, 9),
      timestamp: new Date(),
      message,
      type,
    };
    setEventLogs(prev => [log, ...prev].slice(0, 100)); // Keep last 100 logs
  }, []);

  const connect = useCallback(() => {
    try {
      ws.current = new WebSocket(WS_URL);

      ws.current.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        addLog('Connected to master server', 'success');
      };

      ws.current.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data);
          console.log('Received message:', message);

          // Call registered handlers
          messageHandlers.current.forEach((handler) => {
            handler(message);
          });
        } catch (error) {
          console.error('Error parsing message:', error);
        }
      };

      ws.current.onerror = (error) => {
        console.error('WebSocket error:', error);
        addLog('WebSocket error occurred', 'error');
      };

      ws.current.onclose = () => {
        console.log('WebSocket disconnected');
        setIsConnected(false);
        addLog('Disconnected from master server', 'warning');

        // Attempt to reconnect after 5 seconds
        reconnectTimeout.current = setTimeout(() => {
          addLog('Attempting to reconnect...', 'info');
          connect();
        }, 5000);
      };
    } catch (error) {
      console.error('Error creating WebSocket:', error);
      addLog('Failed to connect to server', 'error');
    }
  }, [addLog]);

  const sendMessage = useCallback((message: WSMessage) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(message));
      console.log('Sent message:', message);
    } else {
      console.error('WebSocket is not connected');
      addLog('Cannot send message: Not connected', 'error');
    }
  }, [addLog]);

  const registerHandler = useCallback((id: string, handler: (data: any) => void) => {
    messageHandlers.current.set(id, handler);
    return () => {
      messageHandlers.current.delete(id);
    };
  }, []);

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [connect]);

  return {
    isConnected,
    sendMessage,
    registerHandler,
    eventLogs,
    addLog,
  };
};
