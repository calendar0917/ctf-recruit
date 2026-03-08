package auth

import (
	"context"
	"testing"
	"time"

	"ctf/backend/internal/runtime"
)

type fakeRepo struct {
	users  map[int64]User
	lookup map[string]int64
	nextID int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		users:  make(map[int64]User),
		lookup: make(map[string]int64),
		nextID: 1,
	}
}

func (r *fakeRepo) CreateUser(_ context.Context, params CreateUserParams) (User, error) {
	id := r.nextID
	r.nextID++
	user := User{
		ID:           id,
		Role:         params.RoleName,
		Username:     params.Username,
		Email:        params.Email,
		DisplayName:  params.DisplayName,
		Status:       "active",
		PasswordHash: params.PasswordHash,
	}
	r.users[id] = user
	r.lookup[params.Username] = id
	r.lookup[params.Email] = id
	return user, nil
}

func (r *fakeRepo) GetUserByIdentifier(_ context.Context, identifier string) (User, error) {
	id, ok := r.lookup[identifier]
	if !ok {
		return User{}, runtime.ErrRepositoryNotFound
	}
	return r.users[id], nil
}

func (r *fakeRepo) GetUserByID(_ context.Context, userID int64) (User, error) {
	user, ok := r.users[userID]
	if !ok {
		return User{}, runtime.ErrRepositoryNotFound
	}
	return user, nil
}

func TestRegisterAndAuthenticate(t *testing.T) {
	repo := newFakeRepo()
	tokens := NewTokenManager("secret", time.Hour)
	service := NewService(repo, tokens)

	result, err := service.Register(context.Background(), RegisterInput{
		Username:    "alice",
		Email:       "alice@example.com",
		Password:    "Password123!",
		DisplayName: "Alice",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if result.Token == "" {
		t.Fatalf("expected token")
	}

	claims, err := service.Authenticate(result.Token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if claims.UserID != result.User.ID {
		t.Fatalf("unexpected user id in token: %d", claims.UserID)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	repo := newFakeRepo()
	hash, err := HashPassword("Password123!")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	_, err = repo.CreateUser(context.Background(), CreateUserParams{
		RoleName:     "player",
		Username:     "alice",
		Email:        "alice@example.com",
		DisplayName:  "Alice",
		PasswordHash: hash,
	})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	service := NewService(repo, NewTokenManager("secret", time.Hour))
	_, err = service.Login(context.Background(), LoginInput{
		Identifier: "alice",
		Password:   "wrong",
	})
	if err != ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}
