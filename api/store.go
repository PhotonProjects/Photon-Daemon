package api

import (
	"sync"

	"github.com/Maj-Studios/Photon-Daemon/internal/server"
)

// MemoryStore est un stockage de serveurs en mémoire.
type MemoryStore struct {
	mu      sync.RWMutex
	servers map[string]*server.Server
}

// NewMemoryStore crée un nouveau stockage en mémoire.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		servers: make(map[string]*server.Server),
	}
}

func (s *MemoryStore) Get(uuid string) *server.Server {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.servers[uuid]
}

func (s *MemoryStore) List() []*server.Server {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*server.Server, 0, len(s.servers))
	for _, sv := range s.servers {
		result = append(result, sv)
	}
	return result
}

func (s *MemoryStore) Add(sv *server.Server) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.servers[sv.ID()] = sv
}

func (s *MemoryStore) Remove(uuid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.servers, uuid)
}
