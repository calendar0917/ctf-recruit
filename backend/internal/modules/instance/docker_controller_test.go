package instance

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type recordedCommand struct {
	name string
	args []string
}

type fakeRunner struct {
	commands []recordedCommand
	outByCmd map[string]string
	errByCmd map[string]error
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	recorded := recordedCommand{name: name, args: append([]string{}, args...)}
	f.commands = append(f.commands, recorded)
	key := name + " " + strings.Join(args, " ")
	if err, ok := f.errByCmd[key]; ok {
		return "", err
	}
	if name == "docker" && len(args) > 0 {
		switch args[0] {
		case "run":
			if out, ok := f.outByCmd["docker run"]; ok {
				return out, nil
			}
		case "inspect":
			if out, ok := f.outByCmd["docker inspect"]; ok {
				return out, nil
			}
		}
	}
	return f.outByCmd[key], nil
}

func TestDockerControllerStartUsesResourceLimitsAndReturnsAccessInfo(t *testing.T) {
	runner := &fakeRunner{outByCmd: map[string]string{}, errByCmd: map[string]error{}}
	runner.outByCmd["docker run"] = "2957581d19fa70655fdd74b633af37ad17c914d252feb3a420f4e55d191c3d38"
	runner.outByCmd["docker inspect"] = "39001"

	controller := NewDockerControllerWithRunner("localhost", runner)
	port := 80
	result, err := controller.Start(context.Background(), RuntimeStartSpec{
		Image:       "nginx:alpine",
		Command:     []string{"nginx", "-g", "daemon off;"},
		ExposedPort: &port,
		Labels: map[string]string{
			"ctf-recruit.instance-id":  "i-1",
			"ctf-recruit.user-id":      "u-1",
			"ctf-recruit.challenge-id": "c-1",
		},
	})
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if result.ContainerID != "2957581d19fa70655fdd74b633af37ad17c914d252feb3a420f4e55d191c3d38" {
		t.Fatalf("expected normalized container id, got %s", result.ContainerID)
	}
	if result.AccessInfo == nil || result.AccessInfo.Port != 39001 {
		t.Fatalf("expected mapped port 39001, got %#v", result.AccessInfo)
	}

	if len(runner.commands) != 2 {
		t.Fatalf("expected 2 docker commands, got %d", len(runner.commands))
	}
	joined := strings.Join(runner.commands[0].args, " ")
	for _, expect := range []string{"--cpus 0.5", "--memory 512m", "--cap-drop ALL", "--security-opt no-new-privileges"} {
		if !strings.Contains(joined, expect) {
			t.Fatalf("expected docker run args to contain %q, got %q", expect, joined)
		}
	}
}

func TestDockerControllerStopRemovesContainer(t *testing.T) {
	runner := &fakeRunner{outByCmd: map[string]string{}, errByCmd: map[string]error{}}
	controller := NewDockerControllerWithRunner("localhost", runner)

	if err := controller.Stop(context.Background(), "container-stop"); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(runner.commands))
	}
	if got := strings.Join(runner.commands[0].args, " "); got != "rm -f container-stop" {
		t.Fatalf("unexpected stop args: %s", got)
	}
}

func TestDockerControllerStartInspectFailureCleansUpContainer(t *testing.T) {
	runner := &fakeRunner{outByCmd: map[string]string{}, errByCmd: map[string]error{}}
	startKey := "docker run"
	containerID := "a95f581d19fa70655fdd74b633af37ad17c914d252feb3a420f4e55d191c3d3"
	inspectKey := "docker inspect --format {{(index (index .NetworkSettings.Ports \"80/tcp\") 0).HostPort}} " + containerID
	runner.outByCmd[startKey] = containerID
	runner.errByCmd[inspectKey] = errors.New("inspect failed")

	controller := NewDockerControllerWithRunner("localhost", runner)
	port := 80
	_, err := controller.Start(context.Background(), RuntimeStartSpec{Image: "nginx:alpine", ExposedPort: &port})
	if err == nil {
		t.Fatal("expected error")
	}

	if len(runner.commands) != 3 {
		t.Fatalf("expected run + inspect + cleanup rm, got %d", len(runner.commands))
	}
	cleanup := strings.Join(runner.commands[2].args, " ")
	if cleanup != "rm -f "+containerID {
		t.Fatalf("expected cleanup remove, got %s", cleanup)
	}
}
