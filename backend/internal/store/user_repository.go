package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"ctf/backend/internal/auth"
	"ctf/backend/internal/runtime"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, params auth.CreateUserParams) (auth.User, error) {
	const query = `
INSERT INTO users (role_id, username, email, password_hash, display_name, status)
SELECT roles.id, $1, $2, $3, $4, 'active'
FROM roles
WHERE roles.name = $5
RETURNING id, username, email, display_name, status
`

	var user auth.User
	err := r.db.QueryRowContext(ctx, query,
		params.Username,
		params.Email,
		params.PasswordHash,
		params.DisplayName,
		params.RoleName,
	).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.Status)
	if err != nil {
		return auth.User{}, fmt.Errorf("create user: %w", err)
	}
	user.Role = params.RoleName
	return user, nil
}

func (r *UserRepository) GetUserByIdentifier(ctx context.Context, identifier string) (auth.User, error) {
	const query = `
SELECT u.id, r.name, u.username, u.email, u.display_name, u.status, u.password_hash
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE lower(u.username) = lower($1) OR lower(u.email) = lower($1)
LIMIT 1
`

	return r.getOne(ctx, query, strings.ToLower(identifier))
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID int64) (auth.User, error) {
	const query = `
SELECT u.id, r.name, u.username, u.email, u.display_name, u.status, u.password_hash
FROM users u
JOIN roles r ON r.id = u.role_id
WHERE u.id = $1
LIMIT 1
`

	return r.getOne(ctx, query, userID)
}

func (r *UserRepository) getOne(ctx context.Context, query string, arg any) (auth.User, error) {
	var user auth.User
	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&user.ID,
		&user.Role,
		&user.Username,
		&user.Email,
		&user.DisplayName,
		&user.Status,
		&user.PasswordHash,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return auth.User{}, runtime.ErrRepositoryNotFound
		}
		return auth.User{}, fmt.Errorf("query user: %w", err)
	}
	return user, nil
}
