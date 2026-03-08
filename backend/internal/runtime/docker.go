package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DockerManager struct {
	apiVersion string
	client     *http.Client
}

type createContainerRequest struct {
	Image        string                    `json:"Image"`
	Env          []string                  `json:"Env,omitempty"`
	Cmd          []string                  `json:"Cmd,omitempty"`
	Labels       map[string]string         `json:"Labels,omitempty"`
	ExposedPorts map[string]struct{}       `json:"ExposedPorts,omitempty"`
	HostConfig   createContainerHostConfig `json:"HostConfig"`
}

type createContainerHostConfig struct {
	AutoRemove   bool                     `json:"AutoRemove"`
	Memory       int64                    `json:"Memory,omitempty"`
	NanoCPUs     int64                    `json:"NanoCpus,omitempty"`
	PortBindings map[string][]portBinding `json:"PortBindings,omitempty"`
}

type portBinding struct {
	HostIP   string `json:"HostIp,omitempty"`
	HostPort string `json:"HostPort,omitempty"`
}

type createContainerResponse struct {
	ID string `json:"Id"`
}

type inspectContainerResponse struct {
	Name            string `json:"Name"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

func NewDockerManager(socketPath string) *DockerManager {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}

	return &DockerManager{
		apiVersion: "v1.41",
		client: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
	}
}

func (m *DockerManager) Start(ctx context.Context, req StartRequest) (StartedContainer, error) {
	portKey := fmt.Sprintf("%d/%s", req.Config.ContainerPort, networkProtocol(req.Config.ExposedProtocol))
	containerName := buildContainerName(req)

	payload := createContainerRequest{
		Image: req.Config.ImageName,
		Env:   flattenEnv(req.Config.Env),
		Cmd:   req.Config.Command,
		Labels: map[string]string{
			"ctf.platform":       "recruit",
			"ctf.challenge_id":   req.Config.ID,
			"ctf.challenge_slug": req.Config.Slug,
			"ctf.user_id":        strconv.FormatInt(req.UserID, 10),
		},
		ExposedPorts: map[string]struct{}{
			portKey: {},
		},
		HostConfig: createContainerHostConfig{
			AutoRemove: true,
			Memory:     int64(req.Config.MemoryLimitMB) * 1024 * 1024,
			NanoCPUs:   int64(req.Config.CPUMilli) * 1_000_000,
			PortBindings: map[string][]portBinding{
				portKey: {{HostIP: "127.0.0.1", HostPort: ""}},
			},
		},
	}

	createPath := fmt.Sprintf("/%s/containers/create?name=%s", m.apiVersion, url.QueryEscape(containerName))
	resp, err := m.request(ctx, http.MethodPost, createPath, payload)
	if err != nil {
		return StartedContainer{}, err
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusCreated); err != nil {
		return StartedContainer{}, err
	}

	var created createContainerResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return StartedContainer{}, fmt.Errorf("decode create container response: %w", err)
	}

	started := false
	defer func() {
		if started {
			return
		}
		_ = m.removeContainer(context.Background(), created.ID)
	}()

	startPath := fmt.Sprintf("/%s/containers/%s/start", m.apiVersion, created.ID)
	resp, err = m.request(ctx, http.MethodPost, startPath, nil)
	if err != nil {
		return StartedContainer{}, err
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusNoContent); err != nil {
		return StartedContainer{}, err
	}
	started = true

	inspected, err := m.inspectContainer(ctx, created.ID, portKey)
	if err != nil {
		return StartedContainer{}, err
	}

	return StartedContainer{
		ContainerID:   created.ID,
		ContainerName: strings.TrimPrefix(inspected.Name, "/"),
		HostIP:        inspected.NetworkSettings.Ports[portKey][0].HostIP,
		HostPort:      mustAtoi(inspected.NetworkSettings.Ports[portKey][0].HostPort),
	}, nil
}

func (m *DockerManager) Stop(ctx context.Context, containerID string) error {
	stopPath := fmt.Sprintf("/%s/containers/%s/stop?t=5", m.apiVersion, containerID)
	resp, err := m.request(ctx, http.MethodPost, stopPath, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified && resp.StatusCode != http.StatusNotFound {
		return expectStatus(resp, http.StatusNoContent, http.StatusNotModified, http.StatusNotFound)
	}

	return m.removeContainer(ctx, containerID)
}

func (m *DockerManager) inspectContainer(ctx context.Context, containerID, portKey string) (inspectContainerResponse, error) {
	inspectPath := fmt.Sprintf("/%s/containers/%s/json", m.apiVersion, containerID)
	resp, err := m.request(ctx, http.MethodGet, inspectPath, nil)
	if err != nil {
		return inspectContainerResponse{}, err
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return inspectContainerResponse{}, err
	}

	var inspected inspectContainerResponse
	if err := json.NewDecoder(resp.Body).Decode(&inspected); err != nil {
		return inspectContainerResponse{}, fmt.Errorf("decode inspect container response: %w", err)
	}

	bindings := inspected.NetworkSettings.Ports[portKey]
	if len(bindings) == 0 {
		return inspectContainerResponse{}, fmt.Errorf("container %s has no published port for %s", containerID, portKey)
	}

	return inspected, nil
}

func (m *DockerManager) removeContainer(ctx context.Context, containerID string) error {
	removePath := fmt.Sprintf("/%s/containers/%s?force=true", m.apiVersion, containerID)
	resp, err := m.request(ctx, http.MethodDelete, removePath, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return expectStatus(resp, http.StatusNoContent, http.StatusNotFound)
}

func (m *DockerManager) request(ctx context.Context, method, path string, payload any) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return nil, fmt.Errorf("encode docker request: %w", err)
		}
		body = buf
	}

	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, body)
	if err != nil {
		return nil, fmt.Errorf("create docker request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker request %s %s: %w", method, path, err)
	}
	return resp, nil
}

func expectStatus(resp *http.Response, expected ...int) error {
	for _, code := range expected {
		if resp.StatusCode == code {
			return nil
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("docker api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func flattenEnv(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return result
}

func buildContainerName(req StartRequest) string {
	replacer := strings.NewReplacer("/", "-", "_", "-", ":", "-", " ", "-")
	base := replacer.Replace(req.Config.Slug)
	return fmt.Sprintf("ctf-%s-u%d-%d", base, req.UserID, time.Now().Unix())
}

func networkProtocol(exposedProtocol string) string {
	switch strings.ToLower(exposedProtocol) {
	case "udp":
		return "udp"
	default:
		return "tcp"
	}
}

func mustAtoi(value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
