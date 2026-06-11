package docker

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	"https://github.com/PhotonProjects/Photon-Panel"
)

// ContainerConfig regroupe tous les paramètres pour créer un container.
type ContainerConfig struct {
	Image      string
	Name       string
	Cmd        []string
	Entrypoint []string
	Env        []string
	Labels     map[string]string

	Binds      []mount.Mount
	Memory     int64
	Swap       int64
	CPUShares  int64
	IOWeight   uint16
	DiskLimit  int64
	PIDLimit   int64

	ExtraHosts []string
	Tmpfs      map[string]string
}

// Create crée un container Docker sans le démarrer.
func Create(ctx context.Context, cfg ContainerConfig) (string, error) {
	c, err := Docker()
	if err != nil {
		return "", err
	}

	cli, ok := c.(*client.Client)
	if !ok {
		return "", fmt.Errorf("docker: unexpected client type")
	}

	hostCfg := &container.HostConfig{
		Mounts:  cfg.Binds,
		DNS:     config.Get().Docker.Network.DNS,
		NetworkMode: container.NetworkMode(config.Get().Docker.Network.Mode),
		Tmpfs:   cfg.Tmpfs,
		Resources: container.Resources{
			Memory:     cfg.Memory,
			MemorySwap: cfg.Swap,
			CPUshares:  cfg.CPUShares,
			BlkioWeight: cfg.IOWeight,
		},
	}

	if cfg.PIDLimit > 0 {
		hostCfg.Resources.PidsLimit = &cfg.PIDLimit
	}

	containerCfg := &container.Config{
		Image:      cfg.Image,
		Cmd:        cfg.Cmd,
		Entrypoint: cfg.Entrypoint,
		Env:        cfg.Env,
		Labels:     cfg.Labels,
		Tty:        true,
		OpenStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	r, err := cli.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, cfg.Name)
	if err != nil {
		return "", fmt.Errorf("docker: failed to create container %s: %w", cfg.Name, err)
	}

	return r.ID, nil
}

// Start démarre un container existant.
func Start(ctx context.Context, containerID string) error {
	c, err := Docker()
	if err != nil {
		return err
	}

	cli, ok := c.(*client.Client)
	if !ok {
		return fmt.Errorf("docker: unexpected client type")
	}

	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("docker: failed to start container %s: %w", containerID, err)
	}

	return nil
}

// Stop arrête un container avec un timeout.
func Stop(ctx context.Context, containerID string, timeout time.Duration) error {
	c, err := Docker()
	if err != nil {
		return err
	}

	cli, ok := c.(*client.Client)
	if !ok {
		return fmt.Errorf("docker: unexpected client type")
	}

	timeoutSec := int(timeout.Seconds())
	opts := container.StopOptions{Timeout: &timeoutSec}
	if err := cli.ContainerStop(ctx, containerID, opts); err != nil {
		if client.IsErrNotFound(err) {
			return nil
		}
		return fmt.Errorf("docker: failed to stop container %s: %w", containerID, err)
	}

	return nil
}

// Remove supprime un container (force si actif).
func Remove(ctx context.Context, containerID string, force, removeVolumes bool) error {
	c, err := Docker()
	if err != nil {
		return err
	}

	cli, ok := c.(*client.Client)
	if !ok {
		return fmt.Errorf("docker: unexpected client type")
	}

	opts := container.RemoveOptions{
		Force:         force,
		RemoveVolumes: removeVolumes,
	}

	if err := cli.ContainerRemove(ctx, containerID, opts); err != nil {
		if client.IsErrNotFound(err) {
			return nil
		}
		return fmt.Errorf("docker: failed to remove container %s: %w", containerID, err)
	}

	return nil
}

// WaitForCondition attend que le container atteigne un état spécifique.
func WaitForCondition(ctx context.Context, containerID string, condition container.WaitCondition) error {
	c, err := Docker()
	if err != nil {
		return err
	}

	cli, ok := c.(*client.Client)
	if !ok {
		return fmt.Errorf("docker: unexpected client type")
	}

	statusCh, errCh := cli.ContainerWait(ctx, containerID, condition)
	select {
	case err := <-errCh:
		return err
	case <-statusCh:
		return nil
	}
}

// Logs retourne les logs d'un container.
func Logs(ctx context.Context, containerID string, tail int) (string, error) {
	c, err := Docker()
	if err != nil {
		return "", err
	}

	cli, ok := c.(*client.Client)
	if !ok {
		return "", fmt.Errorf("docker: unexpected client type")
	}

	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(tail),
	}

	reader, err := cli.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		return "", fmt.Errorf("docker: failed to get logs for %s: %w", containerID, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ResolveContainerTag resolves a container image to a specific tag.
func ResolveContainerTag(ctx context.Context, imageRef string) (string, error) {
	c, err := Docker()
	if err != nil {
		return "", err
	}

	cli, ok := c.(*client.Client)
	if !ok {
		return "", fmt.Errorf("docker: unexpected client type")
	}

	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageRef || tag == imageRef+":latest" {
				return tag, nil
			}
		}
	}

	return imageRef, nil
}
