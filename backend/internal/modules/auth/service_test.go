package auth

import (
	"context"
	apperrors "ctf-recruit/backend/internal/errors"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type mockRepository struct {
	usersByEmail map[string]*User
	usersByID    map[string]*User
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		usersByEmail: map[string]*User{},
		usersByID:    map[string]*User{},
	}
}

func (m *mockRepository) Create(_ context.Context, user *User) error {
	if _, ok := m.usersByEmail[user.Email]; ok {
		return errors.New("duplicate")
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	copied := *user
	m.usersByEmail[user.Email] = &copied
	m.usersByID[user.ID.String()] = &copied
	return nil
}

func (m *mockRepository) GetByEmail(_ context.Context, email string) (*User, error) {
	if u, ok := m.usersByEmail[email]; ok {
		copied := *u
		return &copied, nil
	}
	return nil, nil
}

func (m *mockRepository) GetByID(_ context.Context, id string) (*User, error) {
	if u, ok := m.usersByID[id]; ok {
		copied := *u
		return &copied, nil
	}
	return nil, nil
}

func TestServiceRegisterAndLoginHappyPath(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, "test-secret", time.Hour)

	regResp, err := svc.Register(context.Background(), RegisterRequest{
		Email:       "player@example.com",
		Password:    "password123",
		DisplayName: "Player One",
	})
	if err != nil {
		t.Fatalf("expected register success, got error: %v", err)
	}
	if regResp.Email != "player@example.com" {
		t.Fatalf("expected email to match, got %s", regResp.Email)
	}
	if regResp.Role != RolePlayer {
		t.Fatalf("expected default role player, got %s", regResp.Role)
	}

	stored, _ := repo.GetByEmail(context.Background(), "player@example.com")
	if stored == nil {
		t.Fatal("expected stored user")
	}
	if stored.PasswordHash == "password123" {
		t.Fatal("expected password to be hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("password123")); err != nil {
		t.Fatalf("expected hash to validate: %v", err)
	}

	loginResp, err := svc.Login(context.Background(), LoginRequest{
		Email:    "player@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected login success, got error: %v", err)
	}
	if loginResp.AccessToken == "" {
		t.Fatal("expected access token")
	}
	if loginResp.TokenType != "Bearer" {
		t.Fatalf("expected token type Bearer, got %s", loginResp.TokenType)
	}
}

func TestServiceLoginInvalidCredentials(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, "test-secret", time.Hour)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:       "player@example.com",
		Password:    "password123",
		DisplayName: "Player One",
	})
	if err != nil {
		t.Fatalf("setup register failed: %v", err)
	}

	_, err = svc.Login(context.Background(), LoginRequest{
		Email:    "player@example.com",
		Password: "wrong-pass",
	})
	if err == nil {
		t.Fatal("expected invalid credentials error")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Code != "AUTH_INVALID_CREDENTIALS" {
		t.Fatalf("expected AUTH_INVALID_CREDENTIALS, got %s", appErr.Code)
	}
}
