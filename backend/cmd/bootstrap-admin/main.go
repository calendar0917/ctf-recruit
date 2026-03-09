package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"ctf/backend/internal/auth"
	"ctf/backend/internal/config"
	"ctf/backend/internal/runtime"
	"ctf/backend/internal/store"
)

func main() {
	cfg := config.Load()
	db, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	username := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_USERNAME"))
	email := strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL")))
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	displayName := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_DISPLAY_NAME"))
	if displayName == "" {
		displayName = "Administrator"
	}

	if username == "" || email == "" || password == "" {
		log.Fatal("BOOTSTRAP_ADMIN_USERNAME, BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD are required")
	}

	repo := store.NewUserRepository(db)
	ctx := context.Background()
	if _, err := repo.GetUserByIdentifier(ctx, email); err == nil {
		log.Fatalf("admin bootstrap refused: user already exists for %s", email)
	} else if !errors.Is(err, runtime.ErrRepositoryNotFound) {
		log.Fatalf("check existing admin by email: %v", err)
	}
	if _, err := repo.GetUserByIdentifier(ctx, username); err == nil {
		log.Fatalf("admin bootstrap refused: user already exists for %s", username)
	} else if !errors.Is(err, runtime.ErrRepositoryNotFound) {
		log.Fatalf("check existing admin by username: %v", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	user, err := repo.CreateUser(ctx, auth.CreateUserParams{
		RoleName:     "admin",
		Username:     username,
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: hash,
	})
	if err != nil {
		log.Fatalf("create bootstrap admin: %v", err)
	}

	emit("info", "bootstrap_admin.created", map[string]any{"username": user.Username, "email": user.Email, "id": user.ID})
}

func emit(level, event string, fields map[string]any) {
	entry := map[string]any{
		"level": level,
		"time":  time.Now().UTC().Format(time.RFC3339Nano),
		"event": event,
	}
	for key, value := range fields {
		entry[key] = value
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		log.Printf("structured log fallback: event=%s fields=%v", event, fields)
		return
	}
	log.Print(string(payload))
}
