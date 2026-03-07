package middleware

import (
	apperrors "ctf-recruit/backend/internal/errors"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type rateLimitCounter struct {
	windowStart time.Time
	count       int
}

type RateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	counters map[string]rateLimitCounter
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}

	return &RateLimiter{
		limit:    limit,
		window:   window,
		counters: make(map[string]rateLimitCounter),
	}
}

func (r *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		if ip == "" {
			ip = "unknown"
		}
		key := fmt.Sprintf("%s|%s", c.Path(), ip)

		now := time.Now().UTC()
		allowed, retryAfter := r.allow(key, now)
		if !allowed {
			c.Set("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())+1))
			return apperrors.TooManyRequests("RATE_LIMITED", "Too many requests", fiber.Map{
				"path":       c.Path(),
				"retryAfter": retryAfter.String(),
			})
		}

		return c.Next()
	}
}

func (r *RateLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.counters[key]
	if !exists || now.Sub(entry.windowStart) >= r.window {
		r.counters[key] = rateLimitCounter{windowStart: now, count: 1}
		return true, 0
	}

	if entry.count >= r.limit {
		retryAfter := r.window - now.Sub(entry.windowStart)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	entry.count++
	r.counters[key] = entry
	return true, 0
}
