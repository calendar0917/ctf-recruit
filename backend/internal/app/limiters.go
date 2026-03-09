package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"ctf/backend/internal/config"
)

type AppLimiters struct {
	Register       RateLimiter
	Login          RateLimiter
	Submission     RateLimiter
	AdminWrite     RateLimiter
	RedisAvailable bool
}

func newAppLimiters(cfg config.Config) AppLimiters {
	redisEnabled := strings.TrimSpace(cfg.RedisAddr) != ""
	build := func(scope string, windowSeconds int, max int) RateLimiter {
		window := cfg.RateLimitWindow(scope, windowSeconds)
		fallback := RateLimiter(newMemoryRateLimiter(window, max))
		if !redisEnabled || window <= 0 || max <= 0 {
			return fallback
		}
		primary := newRedisRateLimiter(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.RedisKeyPrefix+"rl:"+scope+":", window, max)
		return newFallbackRateLimiter(primary, fallback, log.Printf)
	}

	return AppLimiters{
		Register:       build("register", cfg.RegisterRateLimitWindowSeconds, cfg.RegisterRateLimitMax),
		Login:          build("login", cfg.LoginRateLimitWindowSeconds, cfg.LoginRateLimitMax),
		Submission:     build("submission", cfg.SubmissionRateLimitWindowSeconds, cfg.SubmissionRateLimitMax),
		AdminWrite:     build("admin_write", cfg.AdminWriteRateLimitWindowSeconds, cfg.AdminWriteRateLimitMax),
		RedisAvailable: redisEnabled,
	}
}

func limitKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func enforceRateLimit(ctx context.Context, limiter RateLimiter, key string) (bool, error) {
	if limiter == nil {
		return true, nil
	}
	return limiter.Allow(ctx, key)
}

func authRateLimitKey(prefix, identifier string, r *http.Request) string {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	if identifier == "" {
		identifier = "anonymous"
	}
	return limitKey(prefix, requestSourceIP(r), identifier)
}

func registerRateLimitKey(r *http.Request) string {
	return limitKey("register", requestSourceIP(r))
}

func adminRateLimitKey(action string, r *http.Request, actorUserID int64) string {
	return limitKey("admin", action, fmt.Sprintf("%d", actorUserID), requestSourceIP(r))
}
