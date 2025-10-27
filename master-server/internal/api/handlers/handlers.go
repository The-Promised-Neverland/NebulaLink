package handlers

import (
	"sync"

	"github.com/The-Promised-Neverland/master-server/internal/service"
)

type Handler struct {
	Service         *service.Service
	PendingRequests map[string]chan interface{}
	Mutex           sync.RWMutex
}

func NewHandler(s *service.Service) *Handler {
	return &Handler{
		Service:         s,
		PendingRequests: make(map[string]chan interface{}),
	}
}
