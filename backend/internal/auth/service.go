package auth

import (
	"context"
	"errors"
	"strings"
	"time"
)

type Repository interface {
	CreateUser(context.Context, CreateUserParams) (User, error)
	GetUserByIdentifier(context.Context, string) (User, error)
	GetUserByID(context.Context, int64) (User, error)
}

type CreateUserParams struct {
	RoleName     string
	Username     string
	Email        string
	DisplayName  string
	PasswordHash string
}

type Service struct {
	repo   Repository
	tokens *TokenManager
}

type AuthResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

func NewService(repo Repository, tokens *TokenManager) *Service {
	return &Service{repo: repo, tokens: tokens}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (AuthResult, error) {
	hash, err := HashPassword(input.Password)
	if err != nil {
		return AuthResult{}, err
	}

	user, err := s.repo.CreateUser(ctx, CreateUserParams{
		RoleName:     "player",
		Username:     strings.TrimSpace(input.Username),
		Email:        strings.ToLower(strings.TrimSpace(input.Email)),
		DisplayName:  strings.TrimSpace(input.DisplayName),
		PasswordHash: hash,
	})
	if err != nil {
		return AuthResult{}, err
	}

	return s.issueToken(user)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (AuthResult, error) {
	user, err := s.repo.GetUserByIdentifier(ctx, strings.TrimSpace(input.Identifier))
	if err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}
	if user.Status != "active" {
		return AuthResult{}, ErrInvalidCredentials
	}
	if err := CheckPassword(user.PasswordHash, input.Password); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	return s.issueToken(user)
}

func (s *Service) Authenticate(token string) (TokenClaims, error) {
	return s.tokens.Verify(token)
}

func (s *Service) Me(ctx context.Context, userID int64) (User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return User{}, err
	}
	if user.Status != "active" {
		return User{}, ErrInvalidCredentials
	}
	return user, nil
}

func (s *Service) issueToken(user User) (AuthResult, error) {
	token, expiresAt, err := s.tokens.Sign(TokenClaims{
		UserID: user.ID,
		Role:   user.Role,
	})
	if err != nil {
		return AuthResult{}, err
	}

	user.PasswordHash = ""
	return AuthResult{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	}, nil
}

func IsTokenError(err error) bool {
	return errors.Is(err, ErrTokenInvalid) || errors.Is(err, ErrTokenExpired)
}
