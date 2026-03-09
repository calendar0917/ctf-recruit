package challengeimport

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ctf/backend/internal/game"
)

func TestParseSpecParsesCurrentTemplateShape(t *testing.T) {
	spec, err := parseSpec(bufio.NewScanner(strings.NewReader(`
meta:
  slug: web-welcome
  title: Welcome Panel
  category: web
  points: 100
  difficulty: easy
  dynamic: true
  visible: true
  sort_order: 10
flag:
  type: static
  value: flag{welcome}
content:
  description: Demo challenge
runtime:
  image: ctf/web-welcome:dev
  mode: per-user
  expose: http
  container_port: 80
  ttl: 30m
  memory_limit_mb: 256
  cpu_limit_millicores: 500
  max_renew_count: 1
  max_active_instances: 5
  user_cooldown: 2m
  env:
    MODE: prod
  command:
    - /app/start
`)))
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	if spec.Meta.Slug != "web-welcome" {
		t.Fatalf("unexpected slug: %q", spec.Meta.Slug)
	}
	if spec.Flag.Type != game.FlagTypeStatic {
		t.Fatalf("unexpected flag type: %q", spec.Flag.Type)
	}
	if spec.Runtime == nil {
		t.Fatal("expected runtime config")
	}
	if spec.Runtime.TTL != 30*time.Minute {
		t.Fatalf("unexpected ttl: %s", spec.Runtime.TTL)
	}
	if spec.Runtime.Env["MODE"] != "prod" {
		t.Fatalf("unexpected env: %+v", spec.Runtime.Env)
	}
	if len(spec.Runtime.Command) != 1 || spec.Runtime.Command[0] != "/app/start" {
		t.Fatalf("unexpected command: %+v", spec.Runtime.Command)
	}
}

func TestNormalizeSpecAppliesDefaultsAndFlagNormalization(t *testing.T) {
	normalized, err := NormalizeSpec(ChallengeSpec{
		Meta: ChallengeMeta{
			Slug:     "demo",
			Title:    "Demo",
			Category: "web",
			Points:   100,
		},
		Flag: ChallengeFlag{Type: " CASE_INSENSITIVE ", Value: "Flag{Demo}"},
	})
	if err != nil {
		t.Fatalf("normalize spec: %v", err)
	}
	if normalized.Meta.Difficulty != "normal" {
		t.Fatalf("expected default difficulty, got %q", normalized.Meta.Difficulty)
	}
	if normalized.Meta.SortOrder != 10 {
		t.Fatalf("expected default sort order, got %d", normalized.Meta.SortOrder)
	}
	if normalized.Flag.Type != game.FlagTypeCaseInsensitive {
		t.Fatalf("expected normalized flag type, got %q", normalized.Flag.Type)
	}
}

func TestNormalizeSpecRejectsUnsupportedRuntimeMode(t *testing.T) {
	_, err := NormalizeSpec(ChallengeSpec{
		Meta: ChallengeMeta{Slug: "demo", Title: "Demo", Category: "web", Points: 100, Dynamic: true},
		Flag: ChallengeFlag{Type: game.FlagTypeStatic, Value: "flag{demo}"},
		Runtime: &ChallengeRuntime{
			ImageName:       "ctf/demo:dev",
			Mode:            "shared",
			ExposedProtocol: "http",
			ContainerPort:   80,
			TTL:             30 * time.Minute,
		},
	})
	if err == nil || !strings.Contains(err.Error(), "only per-user is allowed") {
		t.Fatalf("expected unsupported mode error, got %v", err)
	}
}

func TestDiscoverSpecFilesFindsNestedChallengeYAML(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "templates", "web-a", "challenge.yaml"), "meta:\n  slug: web-a\n")
	mustWriteFile(t, filepath.Join(root, "templates", "web-b", "challenge.yaml"), "meta:\n  slug: web-b\n")
	mustWriteFile(t, filepath.Join(root, "README.md"), "ignore")

	paths, err := DiscoverSpecFiles(root)
	if err != nil {
		t.Fatalf("discover spec files: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 spec files, got %d (%v)", len(paths), paths)
	}
	if !strings.HasSuffix(paths[0], filepath.Join("templates", "web-a", "challenge.yaml")) || !strings.HasSuffix(paths[1], filepath.Join("templates", "web-b", "challenge.yaml")) {
		t.Fatalf("unexpected paths: %+v", paths)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
