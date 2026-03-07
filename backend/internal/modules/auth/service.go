package auth

import (
	"context"
	apperrors "ctf-recruit/backend/internal/errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID string `json:"uid"`
	Role   Role   `json:"role"`
	jwt.RegisteredClaims
}

type Service struct {
	repo      Repository
	jwtSecret []byte
	jwtTTL    time.Duration
}

const (
	defaultAdminListLimit = 50
	maxAdminListLimit     = 200
)

func NewService(repo Repository, jwtSecret string, jwtTTL time.Duration) *Service {
	return &Service{repo: repo, jwtSecret: []byte(jwtSecret), jwtTTL: jwtTTL}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" || strings.TrimSpace(req.DisplayName) == "" {
		return nil, apperrors.BadRequest("AUTH_VALIDATION_ERROR", "Email, password, and display name are required")
	}

	existing, err := s.repo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		return nil, apperrors.Internal("AUTH_REGISTER_FAILED", "Failed to register user", fmt.Errorf("lookup user: %w", err))
	}
	if existing != nil {
		return nil, apperrors.Conflict("AUTH_EMAIL_ALREADY_EXISTS", "Email is already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.Internal("AUTH_REGISTER_FAILED", "Failed to register user", fmt.Errorf("hash password: %w", err))
	}

	user := &User{
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: string(hash),
		DisplayName:  strings.TrimSpace(req.DisplayName),
		Role:         RolePlayer,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		if isDuplicateEmailError(err) {
			return nil, apperrors.Conflict("AUTH_EMAIL_ALREADY_EXISTS", "Email is already registered")
		}
		return nil, apperrors.Internal("AUTH_REGISTER_FAILED", "Failed to register user", fmt.Errorf("create user: %w", err))
	}

	resp := mapUserResponse(user)
	return &resp, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		return nil, apperrors.BadRequest("AUTH_VALIDATION_ERROR", "Email and password are required")
	}

	user, err := s.repo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		return nil, apperrors.Internal("AUTH_LOGIN_FAILED", "Failed to login", fmt.Errorf("lookup user: %w", err))
	}
	if user == nil {
		return nil, apperrors.Unauthorized("AUTH_INVALID_CREDENTIALS", "Invalid email or password")
	}
	if user.IsDisabled {
		return nil, apperrors.Forbidden("AUTH_USER_DISABLED", "User is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperrors.Unauthorized("AUTH_INVALID_CREDENTIALS", "Invalid email or password")
	}

	token, err := s.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	resp := &LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        mapUserResponse(user),
	}
	return resp, nil
}

func (s *Service) Me(ctx context.Context, userID string) (*UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal("AUTH_ME_FAILED", "Failed to fetch user", fmt.Errorf("get user by id: %w", err))
	}
	if user == nil {
		return nil, apperrors.NotFound("AUTH_USER_NOT_FOUND", "User not found")
	}

	resp := mapUserResponse(user)
	return &resp, nil
}

func (s *Service) ListUsers(ctx context.Context, limit, offset int) (*AdminListUsersResponse, error) {
	if limit <= 0 {
		limit = defaultAdminListLimit
	}
	if limit > maxAdminListLimit {
		limit = maxAdminListLimit
	}
	if offset < 0 {
		return nil, apperrors.BadRequest("AUTH_VALIDATION_ERROR", "offset must be zero or greater")
	}

	users, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, apperrors.Internal("AUTH_ADMIN_LIST_USERS_FAILED", "Failed to list users", fmt.Errorf("list users: %w", err))
	}

	items := make([]UserResponse, 0, len(users))
	for i := range users {
		items = append(items, mapUserResponse(&users[i]))
	}

	return &AdminListUsersResponse{Items: items}, nil
}

func (s *Service) AdminUpdateUser(ctx context.Context, operatorID string, targetUserID string, req AdminUpdateUserRequest) (*UserResponse, error) {
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return nil, apperrors.BadRequest("AUTH_VALIDATION_ERROR", "user id is required")
	}

	if req.Role == nil && req.IsDisabled == nil {
		return nil, apperrors.BadRequest("AUTH_VALIDATION_ERROR", "at least one field is required")
	}

	if req.Role != nil && *req.Role != RoleAdmin && *req.Role != RolePlayer {
		return nil, apperrors.BadRequest("AUTH_VALIDATION_ERROR", "role must be admin or player")
	}

	updated, err := s.repo.UpdateAdminFields(ctx, targetUserID, req.Role, req.IsDisabled)
	if err != nil {
		return nil, apperrors.Internal("AUTH_ADMIN_UPDATE_USER_FAILED", "Failed to update user", fmt.Errorf("update user: %w", err))
	}
	if updated == nil {
		return nil, apperrors.NotFound("AUTH_USER_NOT_FOUND", "User not found")
	}

	slog.Info("admin_user_mutation",
		"actorUserId", strings.TrimSpace(operatorID),
		"targetUserId", updated.ID.String(),
		"updatedRole", updated.Role,
		"updatedIsDisabled", updated.IsDisabled,
		"at", time.Now().UTC().Format(time.RFC3339),
	)

	resp := mapUserResponse(updated)
	return &resp, nil
}

func (s *Service) GenerateAccessToken(user *User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: user.ID.String(),
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtTTL)),
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", apperrors.Internal("AUTH_TOKEN_GENERATION_FAILED", "Failed to issue token", fmt.Errorf("sign token: %w", err))
	}

	return tokenString, nil
}

func (s *Service) ParseAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, apperrors.Unauthorized("AUTH_INVALID_TOKEN", "Invalid token")
	}

	return claims, nil
}

func mapUserResponse(user *User) UserResponse {
	return UserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		IsDisabled:  user.IsDisabled,
	}
}

func isDuplicateEmailError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "duplicate key") ||
		strings.Contains(errMsg, "duplicate entry") ||
		strings.Contains(errMsg, "unique constraint") ||
		strings.Contains(errMsg, "unique violation")
}
