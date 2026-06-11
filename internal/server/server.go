package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/docker/docker/client"

	"https://github.com/PhotonProjects/Photon-Panel"
	"github.com/Maj-Studios/Photon-Daemon/internal/egg"
	"github.com/Maj-Studios/Photon-Daemon/internal/remote"
)

// ServerState représente l'état actuel d'un serveur.
type ServerState string

const (
	StatePending    ServerState = "pending"
	StateInstalling ServerState = "installing"
	StateInstalled  ServerState = "installed"
	StateStarting   ServerState = "starting"
	StateRunning    ServerState = "running"
	StateStopping   ServerState = "stopping"
	StateOffline    ServerState = "offline"
	StateSuspended  ServerState = "suspended"
)

// Server représente une instance de serveur de jeu.
type Server struct {
	mu sync.RWMutex

	UUID   string
	Config ServerConfig

	state  atomic.Value

	egg         *egg.Egg
	resolved    *egg.ResolvedEgg
	remote      *remote.Client
	docker      *client.Client

	ctx    context.Context
	cancel context.CancelFunc
}

// ServerConfig est la configuration d'un serveur envoyée par le Panel.
type ServerConfig struct {
	Name              string            `json:"name"`
	Suspended         bool              `json:"suspended"`
	SkipEggScripts    bool              `json:"skip_egg_scripts"`
	Invocation        string            `json:"invocation"`
	EnvVars           map[string]string `json:"environment_variables"`
	Build             BuildConfig       `json:"build"`
	Allocations       Allocations       `json:"allocations"`
	Egg               json.RawMessage  `json:"egg"`
}

// BuildConfig définit les limites de ressources du serveur.
type BuildConfig struct {
	MemoryLimit int64  `json:"memory_limit"`
	Swap        int64  `json:"swap"`
	CPULimit    int64  `json:"cpu_limit"`
	IOWeight    uint16 `json:"io_weight"`
	DiskLimit   int64  `json:"disk_limit"`
	Threads     *int   `json:"threads,omitempty"`
	OOMDisabled bool   `json:"oom_disabled"`
}

// Allocations contient les assignations IP/Port.
type Allocations struct {
	Default    Allocation   `json:"default"`
	Additional []Allocation `json:"additional"`
}

// Allocation est une assignation IP/Port unique.
type Allocation struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

// New crée une nouvelle instance Server.
func New(uuid string, cfg ServerConfig, remoteClient *remote.Client) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		UUID:   uuid,
		Config: cfg,
		remote: remoteClient,
		ctx:    ctx,
		cancel: cancel,
	}

	s.state.Store(string(StatePending))

	return s, nil
}

// ID retourne l'UUID du serveur.
func (s *Server) ID() string { return s.UUID }

// State retourne l'état actuel.
func (s *Server) State() ServerState {
	return ServerState(s.state.Load().(string))
}

// setState définit l'état et déclenche les événements.
func (s *Server) setState(state ServerState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Store(string(state))
}

// Context retourne le context du serveur.
func (s *Server) Context() context.Context { return s.ctx }

// Cancel annule le context du serveur.
func (s *Server) Cancel() {
	if s.cancel != nil {
		s.cancel()
	}
}

// LoadEgg charge et résout l'egg depuis la configuration.
func (s *Server) LoadEgg(userVars map[string]string, selectedImage string) error {
	parsed, err := egg.ParseEgg(s.Config.Egg)
	if err != nil {
		return fmt.Errorf("server: failed to parse egg: %w", err)
	}
	s.egg = parsed

	resolved, err := egg.ResolveEgg(parsed, userVars, selectedImage)
	if err != nil {
		return fmt.Errorf("server: failed to resolve egg: %w", err)
	}
	s.resolved = resolved

	return nil
}

// Resolved retourne l'egg résolu.
func (s *Server) Resolved() *egg.ResolvedEgg {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.resolved
}

// IsInstalling retourne si le serveur est en cours d'installation.
func (s *Server) IsInstalling() bool {
	return s.State() == StateInstalling
}

// IsRunning retourne si le serveur est en cours d'exécution.
func (s *Server) IsRunning() bool {
	st := s.State()
	return st == StateStarting || st == StateRunning
}

// IsSuspended retourne si le serveur est suspendu.
func (s *Server) IsSuspended() bool {
	return s.Config.Suspended
}

// SetSuspended met à jour le statut de suspension.
func (s *Server) SetSuspended(suspended bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.Suspended = suspended
	if suspended {
		s.setState(StateSuspended)
	}
}

// DataDir retourne le chemin du répertoire de données du serveur.
func (s *Server) DataDir() string {
	return fmt.Sprintf("%s/%s", config.Get().System.DataDir, s.UUID)
}
