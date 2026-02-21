package stun

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/The-Promised-Neverland/agent/internal/config"
	"github.com/The-Promised-Neverland/agent/pkg/logger"
	"github.com/pion/stun/v2"
)

type STUNClient struct {
	serverAddr      string
	currentEndpoint string
	mu              sync.RWMutex
	lastQuery       time.Time
}

type EndpointInfo struct {
	PublicEndpoint string
	Changed        bool
}

func NewSTUNserver(cfg *config.Config) *STUNClient {
	serverAddr := cfg.StunServerAddr()
	if serverAddr == "" {
		serverAddr = "stun.l.google.com:19302" // Default to Google STUN
	}
	return &STUNClient{
		serverAddr: serverAddr,
	}
}

func (s *STUNClient) GetCurrentEndpoint() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentEndpoint
}

func (s *STUNClient) QueryEndpoint() (*EndpointInfo, error) {
	conn, err := net.Dial("udp", s.serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial STUN server: %w", err)
	}
	defer conn.Close()
	client, err := stun.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create STUN client: %w", err)
	}
	defer client.Close()
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	var xorAddr stun.XORMappedAddress
	var queryErr error
	err = client.Do(message, func(res stun.Event) {
		if res.Error != nil {
			queryErr = res.Error
			return
		}
		if err := xorAddr.GetFrom(res.Message); err != nil {
			queryErr = fmt.Errorf("failed to get XOR mapped address: %w", err)
			return
		}
	})
	if queryErr != nil {
		return nil, queryErr
	}
	if err != nil {
		return nil, fmt.Errorf("STUN query failed: %w", err)
	}
	endpoint := fmt.Sprintf("%s:%d", xorAddr.IP.String(), xorAddr.Port)
	s.mu.Lock()
	changed := s.currentEndpoint != endpoint
	if changed {
		s.currentEndpoint = endpoint
		logger.Log.Info("STUN endpoint discovered", "endpoint", endpoint)
	}
	s.lastQuery = time.Now()
	s.mu.Unlock()

	return &EndpointInfo{
		PublicEndpoint: endpoint,
		Changed:        changed,
	}, nil
}

func (s *STUNClient) StartPeriodicQuery(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	_, err := s.QueryEndpoint()
	if err != nil {
		logger.Log.Warn("Initial STUN query failed", "err", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := s.QueryEndpoint()
			if err != nil {
				logger.Log.Warn("Periodic STUN query failed", "err", err)
			}
		}
	}
}
