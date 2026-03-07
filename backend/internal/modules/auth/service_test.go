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
	createErr    error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		usersByEmail: map[string]*User{},
		usersByID:    map[string]*User{},
	}
}

func (m *mockRepository) Create(_ context.Context, user *User) error {
	if m.createErr != nil {
		return m.createErr
	}

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

func (m *mockRepository) List(_ context.Context, limit, offset int) ([]User, error) {
	if limit <= 0 {
		return []User{}, nil
	}

	all := make([]User, 0, len(m.usersByID))
	for _, u := range m.usersByID {
		all = append(all, *u)
	}

	if offset >= len(all) {
		return []User{}, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}

	out := make([]User, 0, end-offset)
	for _, item := range all[offset:end] {
		out = append(out, item)
	}
	return out, nil
}

func (m *mockRepository) UpdateAdminFields(_ context.Context, id string, role *Role, isDisabled *bool) (*User, error) {
	u, ok := m.usersByID[id]
	if !ok {
		return nil, nil
	}
	updated := *u
	if role != nil {
		updated.Role = *role
	}
	if isDisabled != nil {
		updated.IsDisabled = *isDisabled
	}
	m.usersByID[id] = &updated
	m.usersByEmail[updated.Email] = &updated
	copyOut := updated
	return &copyOut, nil
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
	if appErr.Status != 401 {
		t.Fatalf("expected 401 status, got %d", appErr.Status)
	}
	if appErr.Code != "AUTH_INVALID_CREDENTIALS" {
		t.Fatalf("expected AUTH_INVALID_CREDENTIALS, got %s", appErr.Code)
	}
}

func TestServiceLoginDisabledUserForbidden(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, "test-secret", time.Hour)

	regResp, err := svc.Register(context.Background(), RegisterRequest{
		Email:       "disabled@example.com",
		Password:    "password123",
		DisplayName: "Disabled User",
	})
	if err != nil {
		t.Fatalf("setup register failed: %v", err)
	}

	disabled := true
	if _, err := repo.UpdateAdminFields(context.Background(), regResp.ID, nil, &disabled); err != nil {
		t.Fatalf("setup disable failed: %v", err)
	}

	_, err = svc.Login(context.Background(), LoginRequest{
		Email:    "disabled@example.com",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected disabled user login to fail")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Status != 403 {
		t.Fatalf("expected 403 status, got %d", appErr.Status)
	}
	if appErr.Code != "AUTH_USER_DISABLED" {
		t.Fatalf("expected AUTH_USER_DISABLED, got %s", appErr.Code)
	}
}

func TestServiceLoginNonExistentUserUnauthorized(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo, "test-secret", time.Hour)

	_, err := svc.Login(context.Background(), LoginRequest{
		Email:    "missing@example.com",
		Password: "any-password",
	})
	if err == nil {
		t.Fatal("expected invalid credentials error")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Status != 401 {
		t.Fatalf("expected 401 status, got %d", appErr.Status)
	}
	if appErr.Code != "AUTH_INVALID_CREDENTIALS" {
		t.Fatalf("expected AUTH_INVALID_CREDENTIALS, got %s", appErr.Code)
	}
}

func TestServiceRegisterValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		req  RegisterRequest
	}{
		{
			name: "missing email",
			req: RegisterRequest{
				Email:       "  ",
				Password:    "password123",
				DisplayName: "Player One",
			},
		},
		{
			name: "missing password",
			req: RegisterRequest{
				Email:       "player@example.com",
				Password:    " ",
				DisplayName: "Player One",
			},
		},
		{
			name: "missing display name",
			req: RegisterRequest{
				Email:       "player@example.com",
				Password:    "password123",
				DisplayName: "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepository()
			svc := NewService(repo, "test-secret", time.Hour)

			_, err := svc.Register(context.Background(), tc.req)
			if err == nil {
				t.Fatal("expected validation error")
			}

			var appErr *apperrors.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected app error, got %T", err)
			}
			if appErr.Status != 400 {
				t.Fatalf("expected 400 status, got %d", appErr.Status)
			}
			if appErr.Code != "AUTH_VALIDATION_ERROR" {
				t.Fatalf("expected AUTH_VALIDATION_ERROR, got %s", appErr.Code)
			}
		})
	}
}

func TestServiceRegisterDuplicateEmailConflict(t *testing.T) {
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

	_, err = svc.Register(context.Background(), RegisterRequest{
		Email:       "Player@Example.com",
		Password:    "password123",
		DisplayName: "Player Two",
	})
	if err == nil {
		t.Fatal("expected duplicate email error")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Status != 409 {
		t.Fatalf("expected 409 status, got %d", appErr.Status)
	}
	if appErr.Code != "AUTH_EMAIL_ALREADY_EXISTS" {
		t.Fatalf("expected AUTH_EMAIL_ALREADY_EXISTS, got %s", appErr.Code)
	}
}

func TestServiceRegisterDuplicateConflictWhenCreateFails(t *testing.T) {
	repo := newMockRepository()
	repo.createErr = errors.New("duplicate key value violates unique constraint")
	svc := NewService(repo, "test-secret", time.Hour)

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:       "player@example.com",
		Password:    "password123",
		DisplayName: "Player One",
	})
	if err == nil {
		t.Fatal("expected duplicate email error")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected app error, got %T", err)
	}
	if appErr.Status != 409 {
		t.Fatalf("expected 409 status, got %d", appErr.Status)
	}
	if appErr.Code != "AUTH_EMAIL_ALREADY_EXISTS" {
		t.Fatalf("expected AUTH_EMAIL_ALREADY_EXISTS, got %s", appErr.Code)
	}
}
