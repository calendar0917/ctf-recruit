package admin

import (
	"errors"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type ScriptResult struct {
	Command    []string `json:"command"`
	DurationMS int64    `json:"duration_ms"`
	ExitCode   int      `json:"exit_code"`
	Stdout     string   `json:"stdout"`
	Stderr     string   `json:"stderr"`
}

func RunScript(ctx context.Context, command []string, timeout time.Duration) (ScriptResult, error) {
	if len(command) == 0 {
		return ScriptResult{}, fmt.Errorf("command is required")
	}
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	started := time.Now()
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		// If the context deadline hit, prefer that error.
		if ctx.Err() != nil {
			err = ctx.Err()
		}
	}

	trim := func(b *bytes.Buffer) string {
		// keep output bounded to reduce log bloat
		const limit = 64 * 1024
		data := b.Bytes()
		if len(data) > limit {
			data = data[len(data)-limit:]
		}
		scanner := bufio.NewScanner(bytes.NewReader(data))
		lines := make([]string, 0)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		return strings.Join(lines, "\n")
	}

	result := ScriptResult{
		Command:    command,
		DurationMS: time.Since(started).Milliseconds(),
		ExitCode:   exitCode,
		Stdout:     trim(&stdout),
		Stderr:     trim(&stderr),
	}
	if err != nil {
		return result, err
	}
	return result, nil
}
