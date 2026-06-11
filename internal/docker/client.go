package docker

import (
	"fmt"
	"sync"

	"github.com/docker/docker/client"
)

var (
	globalClient *client.Client
	once         sync.Once
)

// NewClient crée ou retourne le client Docker singleton.
func NewClient() (*client.Client, error) {
	var err error
	once.Do(func() {
		globalClient, err = client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			err = fmt.Errorf("docker: failed to create client: %w", err)
			return
		}
		_, err = globalClient.Ping(globalClient.DaemonHost())
		if err != nil {
			err = fmt.Errorf("docker: daemon unreachable: %w", err)
		}
	})
	return globalClient, err
}

// Docker retourne le client singleton existant ou une erreur s'il n'est pas initialisé.
func Docker() (*client.Client, error) {
	if globalClient == nil {
		return nil, fmt.Errorf("docker: client not initialized")
	}
	return globalClient, nil
}
