package service

import (
	"github.com/The-Promised-Neverland/master-server/internal/models"
	"github.com/The-Promised-Neverland/master-server/internal/ws"
)

type Service struct {
	Hub *ws.Hub
}

func NewService(Hub *ws.Hub) *Service {
	return &Service{
		Hub: Hub,
	}
}

func (s *Service) GetAllAgents() []*models.AgentInfo {
	s.Hub.Mutex.RLock()
	defer s.Hub.Mutex.RUnlock()
	agents := make([]*models.AgentInfo, 0, len(s.Hub.Connections))
	for id, agent := range s.Hub.Connections {
		// Skip non-agent connections such as the frontend WebSocket,
		// which typically uses a fixed id like "frontend".
		if id == "frontend" || id == "" {
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
	s.Hub.Send(agentID, req)
}

func (s *Service) GetAgent(agentID string) *models.AgentInfo {
	s.Hub.Mutex.RLock()
	defer s.Hub.Mutex.RUnlock()
	agent := s.Hub.Connections[agentID]
	return &models.AgentInfo{
		AgentID:  agentID,
		Name:     agent.Name,
		OS:       agent.OS,
		LastSeen: agent.LastSeen,
	}
}

func (s *Service) RestartAgent(agentID string) {
}

func (s *Service) UninstallAgent(agentID string) {
}
