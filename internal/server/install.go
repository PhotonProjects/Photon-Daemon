package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"

	"https://github.com/PhotonProjects/Photon-Panel"
	photonDocker "github.com/Maj-Studios/Photon-Daemon/internal/docker"
	"github.com/Maj-Studios/Photon-Daemon/internal/remote"
)

// Install démarre le processus d'installation du serveur.
func (s *Server) Install() error {
	return s.install(false)
}

// Reinstall déclenche une réinstallation.
func (s *Server) Reinstall() error {
	if s.IsRunning() {
		return fmt.Errorf("server: cannot reinstall a running server")
	}
	return s.install(true)
}

func (s *Server) install(reinstall bool) error {
	s.setState(StateInstalling)

	// Récupérer le script d'installation depuis le Panel
	script, err := s.remote.GetInstallationScript(s.ctx, s.UUID)
	if err != nil {
		s.setState(StateOffline)
		return fmt.Errorf("server: failed to get install script: %w", err)
	}

	if err := s.internalInstall(script); err != nil {
		// Notifier le Panel de l'échec
		_ = s.remote.SetInstallationStatus(s.ctx, s.UUID, remote.InstallStatusRequest{
			Successful: false,
			Reinstall:  reinstall,
		})
		s.setState(StateOffline)
		return err
	}

	// Notifier le Panel du succès
	if err := s.remote.SetInstallationStatus(s.ctx, s.UUID, remote.InstallStatusRequest{
		Successful: true,
		Reinstall:  reinstall,
	}); err != nil {
		return fmt.Errorf("server: failed to notify panel: %w", err)
	}

	s.setState(StateInstalled)
	return nil
}

func (s *Server) internalInstall(script *remote.InstallationScript) error {
	// Créer le répertoire de données
	if err := os.MkdirAll(s.DataDir(), 0o755); err != nil {
		return fmt.Errorf("server: failed to create data dir: %w", err)
	}

	// Écrire le script d'installation dans un fichier temporaire
	tmpDir := filepath.Join(config.Get().System.TmpDir, s.UUID)
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		return fmt.Errorf("server: failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "install.sh")
	if err := os.WriteFile(scriptPath, []byte(script.Script), 0o644); err != nil {
		return fmt.Errorf("server: failed to write install script: %w", err)
	}

	// Pull l'image d'installation
	if err := photonDocker.PullIfNotExists(s.ctx, script.ContainerImage); err != nil {
		return fmt.Errorf("server: failed to pull install image: %w", err)
	}

	// Nettoyer l'ancien container d'install s'il existe
	_ = photonDocker.Remove(s.ctx, s.UUID+"_installer", true, false)

	// Créer le container d'installation
	limits := config.Get().Docker.InstallerLimits
	memLimit := limits.Memory
	if s.Config.Build.MemoryLimit > memLimit {
		memLimit = s.Config.Build.MemoryLimit
	}

	cpuLimit := limits.CPU
	if s.Config.Build.CPULimit > 0 && s.Config.Build.CPULimit < cpuLimit {
		cpuLimit = s.Config.Build.CPULimit
	}

	tmpfsSize := strconv.Itoa(int(config.Get().App.TmpfsSize))

	containerID, err := photonDocker.Create(s.ctx, photonDocker.ContainerConfig{
		Image:      script.ContainerImage,
		Name:       s.UUID + "_installer",
		Cmd:        []string{script.Entrypoint, "/mnt/install/install.sh"},
		Entrypoint: []string{},
		Env:        s.Resolved().EnvAsSlice(),
		Labels: map[string]string{
			"Service":       "Photon",
			"ContainerType": "server_installer",
		},
		Binds: []mount.Mount{
			{
				Target:   "/mnt/server",
				Source:   s.DataDir(),
				Type:     mount.TypeBind,
				ReadOnly: false,
			},
			{
				Target:   "/mnt/install",
				Source:   tmpDir,
				Type:     mount.TypeBind,
				ReadOnly: false,
			},
		},
		Memory:    memLimit * 1024 * 1024,
		PIDLimit:  0, // Pas de limite PID pour l'install
		Tmpfs:     map[string]string{"/tmp": "rw,exec,nosuid,size=" + tmpfsSize + "M"},
	})
	if err != nil {
		return fmt.Errorf("server: failed to create install container: %w", err)
	}

	// Démarrer le container
	if err := photonDocker.Start(s.ctx, containerID); err != nil {
		return err
	}

	// Attendre la fin de l'installation
	if err := photonDocker.WaitForCondition(s.ctx, containerID, container.WaitConditionNotRunning); err != nil {
		return fmt.Errorf("server: install container failed: %w", err)
	}

	// Nettoyer
	if err := photonDocker.Remove(s.ctx, containerID, true, false); err != nil {
		return err
	}

	return nil
}

// SyncInstallState synchronise l'état d'installation avec le Panel.
func (s *Server) SyncInstallState() error {
	return nil
}

// EnsureDataDirectoryExists crée le répertoire de données s'il n'existe pas.
func (s *Server) EnsureDataDirectoryExists() error {
	if _, err := os.Stat(s.DataDir()); os.IsNotExist(err) {
		return os.MkdirAll(s.DataDir(), 0o755)
	}
	return nil
}

// WaitForStop attend que le serveur s'arrête.
func (s *Server) WaitForStop(ctx context.Context, timeout time.Duration) error {
	// Vérifier l'état périodiquement
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	deadline := time.After(timeout)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("server: timeout waiting for stop")
		case <-ticker.C:
			if s.State() == StateOffline {
				return nil
			}
		}
	}
}
