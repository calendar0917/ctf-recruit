package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"ctf-recruit/backend/internal/modules/auth"
	"ctf-recruit/backend/internal/modules/challenge"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type seedAccount struct {
	email       string
	password    string
	displayName string
	role        auth.Role
}

type seedChallenge struct {
	title              string
	description        string
	category           string
	difficulty         challenge.Difficulty
	mode               challenge.Mode
	runtimeImage       *string
	runtimeCommand     *string
	runtimeExposedPort *int
	points             int
	flag               string
}

func strPtr(v string) *string { return &v }

func intPtr(v int) *int { return &v }

func main() {
	ctx := context.Background()
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	admin, player, generated := loadSeedAccounts()
	if err := seedUsers(ctx, db, []seedAccount{admin, player}); err != nil {
		slog.Error("failed to seed users", "error", err)
		os.Exit(1)
	}

	challenges := seedChallenges()
	if err := seedChallengeData(ctx, db, challenges); err != nil {
		slog.Error("failed to seed challenges", "error", err)
		os.Exit(1)
	}

	if len(generated) > 0 {
		slog.Warn("generated passwords - store them securely and rotate after first login")
		for label, password := range generated {
			slog.Info("seed account password", "account", label, "password", password)
		}
	}

	slog.Info("seed completed")
}

func loadSeedAccounts() (seedAccount, seedAccount, map[string]string) {
	generated := make(map[string]string)

	adminEmail := envOrDefault("SEED_ADMIN_EMAIL", "admin@ctf.local")
	adminDisplay := envOrDefault("SEED_ADMIN_DISPLAY_NAME", "CTF Admin")
	adminPassword := strings.TrimSpace(os.Getenv("SEED_ADMIN_PASSWORD"))
	if adminPassword == "" {
		adminPassword = generatePassword()
		generated["admin"] = adminPassword
	}

	playerEmail := envOrDefault("SEED_PLAYER_EMAIL", "player@ctf.local")
	playerDisplay := envOrDefault("SEED_PLAYER_DISPLAY_NAME", "CTF Player")
	playerPassword := strings.TrimSpace(os.Getenv("SEED_PLAYER_PASSWORD"))
	if playerPassword == "" {
		playerPassword = generatePassword()
		generated["player"] = playerPassword
	}

	admin := seedAccount{
		email:       adminEmail,
		password:    adminPassword,
		displayName: adminDisplay,
		role:        auth.RoleAdmin,
	}

	player := seedAccount{
		email:       playerEmail,
		password:    playerPassword,
		displayName: playerDisplay,
		role:        auth.RolePlayer,
	}

	return admin, player, generated
}

func seedUsers(ctx context.Context, db *gorm.DB, accounts []seedAccount) error {
	for _, account := range accounts {
		if err := createUserIfMissing(ctx, db, account); err != nil {
			return err
		}
	}
	return nil
}

func createUserIfMissing(ctx context.Context, db *gorm.DB, account seedAccount) error {
	var existing auth.User
	if err := db.WithContext(ctx).Where("email = ?", strings.ToLower(account.email)).First(&existing).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else {
		slog.Info("seed user exists", "email", account.email, "role", existing.Role)
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(account.password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user := auth.User{
		Email:        strings.ToLower(account.email),
		PasswordHash: string(hash),
		DisplayName:  strings.TrimSpace(account.displayName),
		Role:         account.role,
	}

	if err := db.WithContext(ctx).Create(&user).Error; err != nil {
		return err
	}

	slog.Info("seed user created", "email", account.email, "role", account.role)
	return nil
}

func seedChallenges() []seedChallenge {
	return []seedChallenge{
		{
			title:       "Welcome Static",
			description: "Find the static flag hidden in the welcome challenge.",
			category:    "intro",
			difficulty:  challenge.DifficultyEasy,
			mode:        challenge.ModeStatic,
			points:      50,
			flag:        "CTF{welcome-static-flag}",
		},
		{
			title:              "Log Trail",
			description:        "Trace the log trail to locate the issued token in this dynamic exercise.",
			category:           "forensics",
			difficulty:         challenge.DifficultyMedium,
			mode:               challenge.ModeDynamic,
			runtimeImage:       strPtr("busybox:1.36"),
			runtimeCommand:     strPtr("httpd -f -p 8080"),
			runtimeExposedPort: intPtr(8080),
			points:             120,
			flag:               "CTF{dynamic-log-trail}",
		},
	}
}

func seedChallengeData(ctx context.Context, db *gorm.DB, challenges []seedChallenge) error {
	for _, item := range challenges {
		if err := createChallengeIfMissing(ctx, db, item); err != nil {
			return err
		}
	}
	return nil
}

func createChallengeIfMissing(ctx context.Context, db *gorm.DB, item seedChallenge) error {
	var existing challenge.Challenge
	if err := db.WithContext(ctx).
		Where("title = ? AND category = ?", item.title, item.category).
		First(&existing).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	} else {
		slog.Info("seed challenge exists", "title", existing.Title, "mode", existing.Mode)
		return nil
	}

	now := time.Now().UTC()
	record := challenge.Challenge{
		Title:              item.title,
		Description:        item.description,
		Category:           item.category,
		Difficulty:         item.difficulty,
		Mode:               item.mode,
		RuntimeImage:       item.runtimeImage,
		RuntimeCommand:     item.runtimeCommand,
		RuntimeExposedPort: item.runtimeExposedPort,
		Points:             item.points,
		FlagHash:           hashFlag(item.flag),
		IsPublished:        true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := db.WithContext(ctx).Create(&record).Error; err != nil {
		return err
	}

	slog.Info("seed challenge created", "title", item.title, "mode", item.mode)
	return nil
}

func hashFlag(flag string) string {
	hash := sha256.Sum256([]byte(flag))
	return hex.EncodeToString(hash[:])
}

func generatePassword() string {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func envOrDefault(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}
