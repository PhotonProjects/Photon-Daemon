package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	"https://github.com/PhotonProjects/Photon-Panel"
)

// EnsureNetwork creates the daemon network if it doesn't exist.
func EnsureNetwork(ctx context.Context) error {
	c, err := Docker()
	if err != nil {
		return err
	}

	cli, ok := c.(*client.Client)
	if !ok {
		return fmt.Errorf("docker: unexpected client type")
	}

	cfg := config.Get()
	name := cfg.Docker.Network.Name

	networks, err := cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return fmt.Errorf("docker: failed to list networks: %w", err)
	}

	for _, nw := range networks {
		if nw.Name == name {
			return nil
		}
	}

	_, err = cli.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: cfg.Docker.Network.Mode,
	})
	if err != nil {
		return fmt.Errorf("docker: failed to create network %s: %w", name, err)
	}

	return nil
}

// RemoveNetwork removes the daemon network.
func RemoveNetwork(ctx context.Context) error {
	c, err := Docker()
	if err != nil {
		return err
	}

	cli, ok := c.(*client.Client)
	if !ok {
		return fmt.Errorf("docker: unexpected client type")
	}

	name := config.Get().Docker.Network.Name
	if err := cli.NetworkRemove(ctx, name); err != nil {
		return fmt.Errorf("docker: failed to remove network %s: %w", name, err)
	}

	return nil
}
