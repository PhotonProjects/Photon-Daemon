package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"https://github.com/PhotonProjects/Photon-Panel"
)

// Client communique avec le Panel Photon.
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// InstallationScript est retourné par le Panel pour l'installation.
type InstallationScript struct {
	ContainerImage string `json:"container_image"`
	Entrypoint     string `json:"entrypoint"`
	Script         string `json:"script"`
}

// ServerConfigurationResponse est la configuration complète d'un serveur.
type ServerConfigurationResponse struct {
	Settings             json.RawMessage       `json:"settings"`
	ProcessConfiguration *json.RawMessage      `json:"process_configuration,omitempty"`
}

// InstallStatusRequest notifie le Panel du statut d'installation.
type InstallStatusRequest struct {
	Successful bool `json:"successful"`
	Reinstall  bool `json:"reinstall"`
}

// ActivityRequest envoie une activité au Panel.
type ActivityRequest struct {
	Event    string `json:"event"`
	Metadata any    `json:"metadata,omitempty"`
}

// NewClient crée un nouveau client Panel.
func NewClient() *Client {
	cfg := config.Get()
	return &Client{
		baseURL:   cfg.Panel.BaseURL,
		authToken: cfg.Panel.AuthToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) request(ctx context.Context, method, path string, body, result any) error {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	var r io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("remote: failed to marshal request: %w", err)
		}
		r = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, r)
	if err != nil {
		return fmt.Errorf("remote: failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PhotonDaemon/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("remote: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remote: unexpected status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("remote: failed to decode response: %w", err)
		}
	}

	return nil
}

// GetServerConfiguration récupère la configuration d'un serveur depuis le Panel.
func (c *Client) GetServerConfiguration(ctx context.Context, serverUUID string) (*ServerConfigurationResponse, error) {
	path := fmt.Sprintf("/api/servers/%s/config", serverUUID)
	var result ServerConfigurationResponse
	if err := c.request(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetInstallationScript récupère le script d'installation d'un serveur.
func (c *Client) GetInstallationScript(ctx context.Context, serverUUID string) (*InstallationScript, error) {
	path := fmt.Sprintf("/api/servers/%s/install", serverUUID)
	var result InstallationScript
	if err := c.request(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SetInstallationStatus notifie le Panel que l'installation est terminée.
func (c *Client) SetInstallationStatus(ctx context.Context, serverUUID string, req InstallStatusRequest) error {
	path := fmt.Sprintf("/api/servers/%s/install/status", serverUUID)
	return c.request(ctx, http.MethodPost, path, req, nil)
}

// PostActivity envoie une activité au Panel.
func (c *Client) PostActivity(ctx context.Context, serverUUID string, req ActivityRequest) error {
	path := fmt.Sprintf("/api/servers/%s/activity", serverUUID)
	return c.request(ctx, http.MethodPost, path, req, nil)
}
