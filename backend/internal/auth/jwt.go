package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type tokenHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type tokenPayload struct {
	Sub  int64  `json:"sub"`
	Role string `json:"role"`
	Exp  int64  `json:"exp"`
	Iat  int64  `json:"iat"`
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
		now:    time.Now,
	}
}

func (m *TokenManager) Sign(claims TokenClaims) (string, time.Time, error) {
	now := m.now().UTC()
	expiresAt := now.Add(m.ttl)

	headerPart, err := encodeTokenPart(tokenHeader{Alg: "HS256", Typ: "JWT"})
	if err != nil {
		return "", time.Time{}, err
	}
	payloadPart, err := encodeTokenPart(tokenPayload{
		Sub:  claims.UserID,
		Role: claims.Role,
		Exp:  expiresAt.Unix(),
		Iat:  now.Unix(),
	})
	if err != nil {
		return "", time.Time{}, err
	}

	signingInput := headerPart + "." + payloadPart
	signature := m.sign(signingInput)
	return signingInput + "." + signature, expiresAt, nil
}

func (m *TokenManager) Verify(token string) (TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return TokenClaims{}, ErrTokenInvalid
	}

	signingInput := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(m.sign(signingInput))) {
		return TokenClaims{}, ErrTokenInvalid
	}

	var payload tokenPayload
	if err := decodeTokenPart(parts[1], &payload); err != nil {
		return TokenClaims{}, ErrTokenInvalid
	}
	if m.now().UTC().Unix() >= payload.Exp {
		return TokenClaims{}, ErrTokenExpired
	}

	return TokenClaims{
		UserID: payload.Sub,
		Role:   payload.Role,
	}, nil
}

func (m *TokenManager) sign(value string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func encodeTokenPart(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal token part: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeTokenPart(part string, target any) error {
	decoded, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(decoded, target)
}
