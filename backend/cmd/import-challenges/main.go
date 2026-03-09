package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"ctf/backend/internal/challengeimport"
	"ctf/backend/internal/config"
	"ctf/backend/internal/store"
)

func main() {
	cfg := config.Load()

	var (
		contestSlug = flag.String("contest", envOrDefault("IMPORT_CONTEST_SLUG", "recruit-2025"), "contest slug to receive imported challenges")
		root        = flag.String("root", envOrDefault("CHALLENGE_ROOT", detectDefaultRoot()), "directory to scan for challenge.yaml files")
		path        = flag.String("path", strings.TrimSpace(os.Getenv("CHALLENGE_SPEC_PATH")), "optional single challenge.yaml path to import")
	)
	flag.Parse()

	db, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	importer := challengeimport.New(db)
	ctx := context.Background()

	paths := make([]string, 0)
	if strings.TrimSpace(*path) != "" {
		paths = append(paths, strings.TrimSpace(*path))
	} else {
		paths, err = challengeimport.DiscoverSpecFiles(strings.TrimSpace(*root))
		if err != nil {
			log.Fatalf("discover challenge specs: %v", err)
		}
	}
	if len(paths) == 0 {
		log.Fatalf("no challenge specs found under %s", strings.TrimSpace(*root))
	}

	for _, specPath := range paths {
		result, err := importer.ImportFile(ctx, strings.TrimSpace(*contestSlug), specPath)
		if err != nil {
			log.Fatalf("import %s: %v", specPath, err)
		}
		fmt.Printf("imported %s (id=%d runtime=%t) from %s\n", result.Slug, result.ChallengeID, result.RuntimeSynced, result.Path)
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func detectDefaultRoot() string {
	candidates := []string{"../challenges", "./challenges", "/app/challenges"}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return "../challenges"
}
