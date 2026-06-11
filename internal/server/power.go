package server

import (
	"fmt"
	"time"

	"github.com/docker/docker/api/types/mount"

	"https://github.com/PhotonProjects/Photon-Panel"
	"github.com/Maj-Studios/Photon-Daemon/internal/egg"
	photonDocker "github.com/Maj-Studios/Photon-Daemon/internal/docker"
)

// PowerAction représente une action de contrôle du serveur.
type PowerAction string

const (
	PowerStart   PowerAction = "start"
	PowerStop    PowerAction = "stop"
	PowerRestart PowerAction = "restart"
	PowerKill    PowerAction = "kill"
)

// HandlePower exécute une action de contrôle sur le serveur.
func (s *Server) HandlePower(action PowerAction) error {
	resolved := s.Resolved()
	if resolved == nil {
		return fmt.Errorf("server: egg not resolved")
	}

	switch action {
	case PowerStart:
		return s.start(resolved)
	case PowerStop:
		return s.stop()
	case PowerRestart:
		if err := s.stop(); err != nil {
			return err
		}
		return s.start(resolved)
	case PowerKill:
		return s.kill()
	default:
		return fmt.Errorf("server: unknown power action %q", action)
	}
}

func (s *Server) start(resolved *egg.ResolvedEgg) error {
	s.setState(StateStarting)

	// Nettoyer l'ancien container s'il existe
	_ = photonDocker.Remove(s.ctx, s.UUID, true, false)

	// Pull l'image
	if err := photonDocker.PullIfNotExists(s.ctx, resolved.DockerImage); err != nil {
		s.setState(StateOffline)
		return fmt.Errorf("server: failed to pull image: %w", err)
	}

	// Créer le container
	containerID, err := photonDocker.Create(s.ctx, photonDocker.ContainerConfig{
		Image:      resolved.DockerImage,
		Name:       s.UUID,
		Cmd:        []string{resolved.ResolvedStartup},
		Env:        resolved.EnvAsSlice(),
		Labels: map[string]string{
			"Service":       "Photon",
			"ContainerType": "server",
		},
		Binds: []mount.Mount{
			{
				Target:   "/home/container",
				Source:   s.DataDir(),
				Type:     mount.TypeBind,
				ReadOnly: false,
			},
		},
		Memory:   s.Config.Build.MemoryLimit * 1024 * 1024,
		Swap:     s.Config.Build.Swap * 1024 * 1024,
		IOWeight: uint16(s.Config.Build.IOWeight),
		PIDLimit: config.Get().App.ContainerPIDLimit,
	})
	if err != nil {
		s.setState(StateOffline)
		return fmt.Errorf("server: failed to create container: %w", err)
	}

	// Démarrer
	if err := photonDocker.Start(s.ctx, containerID); err != nil {
		s.setState(StateOffline)
		return fmt.Errorf("server: failed to start container: %w", err)
	}

	s.setState(StateRunning)
	return nil
}

func (s *Server) stop() error {
	s.setState(StateStopping)

	if err := photonDocker.Stop(s.ctx, s.UUID, 30*time.Second); err != nil {
		return fmt.Errorf("server: failed to stop: %w", err)
	}

	_ = photonDocker.Remove(s.ctx, s.UUID, false, false)

	s.setState(StateOffline)
	return nil
}

func (s *Server) kill() error {
	if err := photonDocker.Remove(s.ctx, s.UUID, true, false); err != nil {
		return fmt.Errorf("server: failed to kill: %w", err)
	}

	s.setState(StateOffline)
	return nil
}
