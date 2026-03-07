package instance

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

const (
	dockerCPULimit    = "0.5"
	dockerMemoryLimit = "512m"
)

type DockerController struct {
	runner     dockerCommandRunner
	accessHost string
}

type dockerCommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("%s %v failed: %w: %s", name, args, err, trimmed)
	}
	return trimmed, nil
}

func NewDockerController(accessHost string) *DockerController {
	return NewDockerControllerWithRunner(accessHost, execRunner{})
}

func NewDockerControllerWithRunner(accessHost string, runner dockerCommandRunner) *DockerController {
	if strings.TrimSpace(accessHost) == "" {
		accessHost = "localhost"
	}
	if runner == nil {
		runner = execRunner{}
	}
	return &DockerController{runner: runner, accessHost: accessHost}
}

func (c *DockerController) Start(ctx context.Context, spec RuntimeStartSpec) (*RuntimeStartResult, error) {
	image := strings.TrimSpace(spec.Image)
	if image == "" {
		return nil, fmt.Errorf("runtime image is required")
	}

	args := []string{
		"run",
		"--detach",
		"--rm",
		"--cpus", dockerCPULimit,
		"--memory", dockerMemoryLimit,
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
	}
	labelKeys := make([]string, 0, len(spec.Labels))
	for k := range spec.Labels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)
	for _, k := range labelKeys {
		v := spec.Labels[k]
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	if spec.ExposedPort != nil && *spec.ExposedPort > 0 {
		args = append(args, "-p", fmt.Sprintf("127.0.0.1::%d", *spec.ExposedPort))
	}

	args = append(args, image)
	if len(spec.Command) > 0 {
		args = append(args, spec.Command...)
	}

	containerID, err := c.runner.Run(ctx, "docker", args...)
	if err != nil {
		return nil, err
	}
	containerID = normalizeContainerID(containerID)
	if containerID == "" {
		return nil, fmt.Errorf("docker run returned empty container id")
	}

	result := &RuntimeStartResult{ContainerID: containerID}
	if spec.ExposedPort != nil && *spec.ExposedPort > 0 {
		hostPort, inspectErr := c.lookupHostPort(ctx, containerID, *spec.ExposedPort)
		if inspectErr != nil {
			_ = c.Stop(ctx, containerID)
			return nil, inspectErr
		}
		result.AccessInfo = &RuntimeAccessInfo{
			Host:             c.accessHost,
			Port:             hostPort,
			ConnectionString: fmt.Sprintf("%s:%d", c.accessHost, hostPort),
		}
	}

	return result, nil
}

func (c *DockerController) Stop(ctx context.Context, containerID string) error {
	containerID = strings.TrimSpace(containerID)
	if containerID == "" {
		return nil
	}
	_, err := c.runner.Run(ctx, "docker", "rm", "-f", containerID)
	return err
}

func (c *DockerController) lookupHostPort(ctx context.Context, containerID string, exposedPort int) (int, error) {
	key := fmt.Sprintf("%d/tcp", exposedPort)
	tmpl := fmt.Sprintf("{{(index (index .NetworkSettings.Ports %q) 0).HostPort}}", key)
	out, err := c.runner.Run(ctx, "docker", "inspect", "--format", tmpl, containerID)
	if err != nil {
		return 0, err
	}
	port, convErr := strconv.Atoi(strings.TrimSpace(out))
	if convErr != nil {
		return 0, fmt.Errorf("parse mapped host port: %w", convErr)
	}
	return port, nil
}

func normalizeContainerID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if isHexString(raw) && len(raw) >= 12 {
		return raw
	}
	lines := strings.Split(raw, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if isHexString(line) && len(line) >= 12 {
			return line
		}
	}
	return ""
}

func isHexString(s string) bool {
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			continue
		}
		return false
	}
	return true
}
