package service

import (
	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/The-Promised-Neverland/master-server/internal/sse"
	"github.com/The-Promised-Neverland/master-server/internal/ws"
)

type Service struct {
	WSHub  *ws.WSHub
	SSEHub *sse.SSEHub
}

func NewService(wsHub *ws.WSHub, sseHub *sse.SSEHub) *Service {
	return &Service{
		WSHub:  wsHub,
		SSEHub: sseHub,
	}
}

func (s *Service) GetAllAgents() []*models.AgentInfo {
	s.WSHub.Mutex.RLock()
	defer s.WSHub.Mutex.RUnlock()
	agents := make([]*models.AgentInfo, 0, len(s.WSHub.Connections))
	for id, agent := range s.WSHub.Connections {
		if agent.Name == "frontend" || id == "" {
			continue
		}
		info := &models.AgentInfo{
			AgentID:  id,
			Name:     agent.Name,
			OS:       agent.OS,
			LastSeen: agent.LastSeen,
		}
		agents = append(agents, info)
	}
	return agents
}

func (s *Service) TriggerAgentforMetrics(agentID string) {
	req := models.Message{
		Type: "master_metrics_request",
		Payload: nil,
	}
	s.WSHub.Send(agentID, req)
}

func (s *Service) GetAgent(agentID string) *models.AgentInfo {
	s.WSHub.Mutex.RLock()
	defer s.WSHub.Mutex.RUnlock()
	agent := s.WSHub.Connections[agentID]
	if agent == nil {
		return nil
	}
	return &models.AgentInfo{
		AgentID:  agentID,
		Name:     agent.Name,
		OS:       agent.OS,
		LastSeen: agent.LastSeen,
	}
}

func (s *Service) SendAgentListToFrontend() {
	agents := s.GetAllAgents()
	msg := models.Message{
		Type:    "agent_list",
		Payload: agents,
	}
	s.SSEHub.Broadcast(msg)
}

func (s *Service) RestartAgent(agentID string) {
}

func (s *Service) UninstallAgent(agentID string) {
}
