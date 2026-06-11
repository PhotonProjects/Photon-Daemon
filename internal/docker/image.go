package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"

	"https://github.com/PhotonProjects/Photon-Panel"
)

// Pull pulls an image from a registry. Uses registry auth from config if available.
func Pull(ctx context.Context, imageRef string) error {
	c, err := Docker()
	if err != nil {
		return err
	}

	cli, ok := c.(*client.Client)
	if !ok {
		return fmt.Errorf("docker: unexpected client type")
	}

	auth := registryAuth(imageRef)
	opts := image.PullOptions{}
	if auth != "" {
		opts.RegistryAuth = auth
	}

	r, err := cli.ImagePull(ctx, imageRef, opts)
	if err != nil {
		return fmt.Errorf("docker: failed to pull image %s: %w", imageRef, err)
	}
	defer r.Close()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
	}
	return scanner.Err()
}

// PullIfNotExists pulls an image only if it doesn't exist locally.
func PullIfNotExists(ctx context.Context, imageRef string) error {
	c, err := Docker()
	if err != nil {
		return err
	}

	cli, ok := c.(*client.Client)
	if !ok {
		return fmt.Errorf("docker: unexpected client type")
	}

	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return fmt.Errorf("docker: failed to list images: %w", err)
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageRef {
				return nil
			}
		}
	}

	return Pull(ctx, imageRef)
}

func registryAuth(imageRef string) string {
	registries := config.Get().Docker.Registries
	if registries == nil {
		return ""
	}

	for registry, auth := range registries {
		if strings.HasPrefix(imageRef, registry) {
			b64, err := registryAuthBase64(&auth)
			if err != nil {
				return ""
			}
			return b64
		}
	}
	return ""
}

func registryAuthBase64(auth *config.RegistryAuth) (string, error) {
	authStr := fmt.Sprintf("%s:%s", auth.Username, auth.Password)
	return base64.StdEncoding.EncodeToString([]byte(authStr)), nil
}
