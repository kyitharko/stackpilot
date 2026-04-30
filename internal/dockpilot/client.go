// Package dockpilot provides a typed HTTP client for the dockpilot REST API.
// stackpilot calls this client for all container execution; it never touches
// the Docker SDK directly.
package dockpilot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a typed HTTP client for the dockpilot REST API.
type Client struct {
	baseURL string
	http    *http.Client
}

// New returns a Client targeting baseURL (e.g. "http://127.0.0.1:8088").
func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 5 * time.Minute}, // long for image pulls
	}
}

// DeployRequest is the JSON body sent to POST /v1/services/{service}/deploy.
type DeployRequest struct {
	Image   string   `json:"image,omitempty"`
	Ports   []string `json:"ports,omitempty"`
	Volumes []string `json:"volumes,omitempty"`
	Env     []string `json:"env,omitempty"`
	Command []string `json:"command,omitempty"`
}

// DeployResult is the JSON response from a successful deploy.
type DeployResult struct {
	Name      string   `json:"name"`
	Container string   `json:"container"`
	Image     string   `json:"image"`
	Ports     []string `json:"ports"`
}

// ServiceStatus is the JSON response from GET /v1/services/{service}/status.
type ServiceStatus struct {
	Name      string `json:"name"`
	Container string `json:"container"`
	Image     string `json:"image"`
	State     string `json:"state"`
	Ports     string `json:"ports"`
	Running   bool   `json:"running"`
}

// Health calls GET /health and returns an error if the daemon is unreachable.
func (c *Client) Health(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodGet, "/health", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.checkStatus(resp)
}

// Deploy calls POST /v1/services/{service}/deploy with the given request body.
func (c *Client) Deploy(ctx context.Context, service string, req DeployRequest) (*DeployResult, error) {
	resp, err := c.do(ctx, http.MethodPost,
		fmt.Sprintf("/v1/services/%s/deploy", service), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkStatus(resp); err != nil {
		return nil, err
	}

	var result DeployResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding deploy response: %w", err)
	}
	return &result, nil
}

// Remove calls DELETE /v1/services/{service}.
// volumes is an optional comma-separated list of named volumes to also delete.
func (c *Client) Remove(ctx context.Context, service string, volumes []string) error {
	path := fmt.Sprintf("/v1/services/%s", service)
	if len(volumes) > 0 {
		path += "?volumes=" + strings.Join(volumes, ",")
	}
	resp, err := c.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return c.checkStatus(resp)
}

// Status calls GET /v1/services/{service}/status.
func (c *Client) Status(ctx context.Context, service string) (*ServiceStatus, error) {
	resp, err := c.do(ctx, http.MethodGet,
		fmt.Sprintf("/v1/services/%s/status", service), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkStatus(resp); err != nil {
		return nil, err
	}

	var status ServiceStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decoding status response: %w", err)
	}
	return &status, nil
}

// --- internal helpers --------------------------------------------------------

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshalling request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling dockpilot %s %s: %w", method, path, err)
	}
	return resp, nil
}

// checkStatus reads the response body on non-2xx status and returns an error.
func (c *Client) checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)

	var apiErr struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
		return fmt.Errorf("dockpilot API error (%d %s): %s", resp.StatusCode, apiErr.Code, apiErr.Error)
	}
	return fmt.Errorf("dockpilot returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
