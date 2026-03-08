package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"

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

	log.Printf("bootstrap admin created: username=%s email=%s id=%d", user.Username, user.Email, user.ID)
}
