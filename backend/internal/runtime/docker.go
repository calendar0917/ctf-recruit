package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DockerManager struct {
	apiVersion string
	client     *http.Client
}

type DockerManagerConfig struct {
	BindAddr string
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
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		Status string `json:"Status"`
	} `json:"State"`
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

type listContainerSummary struct {
	ID     string            `json:"Id"`
	Labels map[string]string `json:"Labels"`
}

func NewDockerManager(socketPath string) *DockerManager {
	return NewDockerManagerWithConfig(socketPath, DockerManagerConfig{})
}

func NewDockerManagerWithConfig(socketPath string, cfg DockerManagerConfig) *DockerManager {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}

	_ = cfg

	return &DockerManager{
		apiVersion: strings.TrimSpace(os.Getenv("DOCKER_API_VERSION")),
		client: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
	}
}

func (m *DockerManager) Start(ctx context.Context, req StartRequest) (StartedContainer, error) {
	portKey := fmt.Sprintf("%d/%s", req.Config.ContainerPort, networkProtocol(req.Config.ExposedProtocol))
	containerName := buildContainerName(req)

	bindAddr := strings.TrimSpace(req.BindAddr)
	if bindAddr == "" {
		bindAddr = "127.0.0.1"
	}
	hostPort := ""
	if req.HostPort > 0 {
		hostPort = strconv.Itoa(req.HostPort)
	}

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
				portKey: {{HostIP: bindAddr, HostPort: hostPort}},
			},
		},
	}

	createPath := m.apiPath(fmt.Sprintf("/containers/create?name=%s", url.QueryEscape(containerName)))
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

	startPath := m.apiPath(fmt.Sprintf("/containers/%s/start", created.ID))
	resp, err = m.request(ctx, http.MethodPost, startPath, nil)
	if err != nil {
		return StartedContainer{}, err
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusNoContent); err != nil {
		return StartedContainer{}, err
	}
	started = true

	inspected, err := m.inspectContainer(ctx, created.ID)
	if err != nil {
		return StartedContainer{}, err
	}

	bindings := inspected.NetworkSettings.Ports[portKey]
	if len(bindings) == 0 {
		return StartedContainer{}, fmt.Errorf("container %s has no published port for %s", created.ID, portKey)
	}
	if req.HostPort > 0 && mustAtoi(bindings[0].HostPort) != req.HostPort {
		return StartedContainer{}, fmt.Errorf("container %s published port %s does not match requested %d", created.ID, bindings[0].HostPort, req.HostPort)
	}

	return StartedContainer{
		ContainerID:   created.ID,
		ContainerName: strings.TrimPrefix(inspected.Name, "/"),
		HostIP:        bindings[0].HostIP,
		HostPort:      mustAtoi(bindings[0].HostPort),
	}, nil
}

func (m *DockerManager) Stop(ctx context.Context, containerID string) error {
	stopPath := m.apiPath(fmt.Sprintf("/containers/%s/stop?t=5", containerID))
	resp, err := m.request(ctx, http.MethodPost, stopPath, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return expectStatus(resp, http.StatusNoContent, http.StatusNotModified, http.StatusNotFound)
}

func (m *DockerManager) Exists(ctx context.Context, containerID string) (bool, error) {
	inspect, err := m.inspectContainer(ctx, containerID)
	if err != nil {
		if isDockerStatus(err, http.StatusNotFound) {
			return false, nil
		}
		return false, err
	}
	return inspect.State.Status != "removing", nil
}

func (m *DockerManager) ListManagedContainers(ctx context.Context) ([]ManagedContainer, error) {
	filters := url.QueryEscape(`{"label":["ctf.platform=recruit"]}`)
	path := m.apiPath("/containers/json?all=true&filters=" + filters)
	resp, err := m.request(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var items []listContainerSummary
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode list containers response: %w", err)
	}

	result := make([]ManagedContainer, 0, len(items))
	for _, item := range items {
		userID, err := strconv.ParseInt(strings.TrimSpace(item.Labels["ctf.user_id"]), 10, 64)
		if err != nil {
			continue
		}
		challengeID := strings.TrimSpace(item.Labels["ctf.challenge_id"])
		if challengeID == "" {
			continue
		}
		result = append(result, ManagedContainer{
			ContainerID: item.ID,
			ChallengeID: challengeID,
			UserID:      userID,
		})
	}
	return result, nil
}

func (m *DockerManager) inspectContainer(ctx context.Context, containerID string) (inspectContainerResponse, error) {
	inspectPath := m.apiPath(fmt.Sprintf("/containers/%s/json", containerID))
	resp, err := m.request(ctx, http.MethodGet, inspectPath, nil)
	if err != nil {
		return inspectContainerResponse{}, err
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK, http.StatusNotFound); err != nil {
		return inspectContainerResponse{}, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return inspectContainerResponse{}, dockerStatusError{statusCode: http.StatusNotFound, message: "not found"}
	}

	var inspected inspectContainerResponse
	if err := json.NewDecoder(resp.Body).Decode(&inspected); err != nil {
		return inspectContainerResponse{}, fmt.Errorf("decode inspect container response: %w", err)
	}
	return inspected, nil
}

func (m *DockerManager) removeContainer(ctx context.Context, containerID string) error {
	removePath := m.apiPath(fmt.Sprintf("/containers/%s?force=true", containerID))
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

func (m *DockerManager) apiPath(path string) string {
	if m.apiVersion == "" {
		return path
	}
	return "/" + strings.TrimPrefix(m.apiVersion, "/") + path
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

type dockerStatusError struct {
	statusCode int
	message    string
}

func (e dockerStatusError) Error() string {
	return fmt.Sprintf("docker api returned %d: %s", e.statusCode, e.message)
}

func isDockerStatus(err error, statusCode int) bool {
	var target dockerStatusError
	if !errors.As(err, &target) {
		return false
	}
	return target.statusCode == statusCode
}

func expectStatus(resp *http.Response, expected ...int) error {
	for _, code := range expected {
		if resp.StatusCode == code {
			return nil
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return dockerStatusError{statusCode: resp.StatusCode, message: strings.TrimSpace(string(body))}
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
