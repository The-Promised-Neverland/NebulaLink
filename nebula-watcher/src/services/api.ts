import { env } from "@/config/env";
import type {
  HealthCheckResponse,
  AgentListResponse,
  AgentInfoResponse,
  MetricsPayload,
  ActionResponse,
  Message,
} from "@/types";

class ApiService {
  private baseUrl: string;

  constructor() {
    this.baseUrl = env.apiBaseUrl;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    
    const response = await fetch(url, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...options.headers,
      },
    });

    if (!response.ok) {
      throw new Error(`API Error: ${response.status} ${response.statusText}`);
    }

    return response.json();
  }

  // Health Check
  async getHealth(): Promise<HealthCheckResponse> {
    return this.request<HealthCheckResponse>("/health");
  }

  // List All Agents
  async getAgents(): Promise<AgentListResponse> {
    return this.request<AgentListResponse>("/api/v1/agents");
  }

  // Get Agent Details
  async getAgent(id: string): Promise<AgentInfoResponse> {
    return this.request<AgentInfoResponse>(`/api/v1/agents/${encodeURIComponent(id)}`);
  }

  // Get Agent Metrics
  async getAgentMetrics(id: string): Promise<Message<MetricsPayload>> {
    return this.request<Message<MetricsPayload>>(
      `/api/v1/agents/${encodeURIComponent(id)}/metrics`
    );
  }

  // Restart Agent
  async restartAgent(id: string): Promise<ActionResponse> {
    return this.request<ActionResponse>(
      `/api/v1/agents/${encodeURIComponent(id)}/restart`,
      { method: "POST" }
    );
  }

  // Uninstall Agent
  async uninstallAgent(id: string): Promise<ActionResponse> {
    return this.request<ActionResponse>(
      `/api/v1/agents/${encodeURIComponent(id)}/uninstall`,
      { method: "POST" }
    );
  }

  // Request File/Folder from Agent
  async requestFileSystem(
    requestingAgentId: string,
    sourceAgentId: string,
    path: string
  ): Promise<ActionResponse> {
    const url = `/api/v1/agents/${encodeURIComponent(requestingAgentId)}/filesystem/${encodeURIComponent(sourceAgentId)}`;
    const response = await fetch(`${this.baseUrl}${url}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ path }),
    });

    if (!response.ok) {
      throw new Error(`API Error: ${response.status} ${response.statusText}`);
    }

    return response.json();
  }
}

export const api = new ApiService();
