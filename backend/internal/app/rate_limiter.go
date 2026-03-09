package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RateLimiter interface {
	Allow(context.Context, string) (bool, error)
}

type memoryRateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	window  time.Duration
	max     int
	buckets map[string]rateLimitBucket
}

type rateLimitBucket struct {
	count     int
	windowEnd time.Time
}

func newMemoryRateLimiter(window time.Duration, max int) *memoryRateLimiter {
	return &memoryRateLimiter{
		now:     time.Now,
		window:  window,
		max:     max,
		buckets: make(map[string]rateLimitBucket),
	}
}

func (l *memoryRateLimiter) Allow(_ context.Context, key string) (bool, error) {
	if l == nil || l.window <= 0 || l.max <= 0 {
		return true, nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().UTC()
	bucket, ok := l.buckets[key]
	if !ok || !now.Before(bucket.windowEnd) {
		l.buckets[key] = rateLimitBucket{count: 1, windowEnd: now.Add(l.window)}
		return true, nil
	}
	if bucket.count >= l.max {
		return false, nil
	}
	bucket.count++
	l.buckets[key] = bucket
	return true, nil
}

type redisRateLimiter struct {
	addr        string
	password    string
	db          int
	prefix      string
	window      time.Duration
	max         int
	dialTimeout time.Duration
}

func newRedisRateLimiter(addr, password string, db int, prefix string, window time.Duration, max int) *redisRateLimiter {
	return &redisRateLimiter{
		addr:        addr,
		password:    password,
		db:          db,
		prefix:      prefix,
		window:      window,
		max:         max,
		dialTimeout: 2 * time.Second,
	}
}

func (l *redisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if l == nil || l.window <= 0 || l.max <= 0 {
		return true, nil
	}

	conn, reader, err := l.open(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	fullKey := l.prefix + key
	count, err := l.sendIntCommand(conn, reader, "INCR", fullKey)
	if err != nil {
		return false, err
	}
	if count == 1 {
		if _, err := l.sendIntCommand(conn, reader, "PEXPIRE", fullKey, strconv.FormatInt(l.window.Milliseconds(), 10)); err != nil {
			return false, err
		}
	}
	return count <= l.max, nil
}

func (l *redisRateLimiter) open(ctx context.Context) (net.Conn, *bufio.Reader, error) {
	dialer := &net.Dialer{Timeout: l.dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", l.addr)
	if err != nil {
		return nil, nil, fmt.Errorf("dial redis: %w", err)
	}

	reader := bufio.NewReader(conn)
	if err := l.applySession(ctx, conn, reader); err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	return conn, reader, nil
}

func (l *redisRateLimiter) applySession(ctx context.Context, conn net.Conn, reader *bufio.Reader) error {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(l.dialTimeout))
	}
	if l.password != "" {
		if _, err := l.sendSimpleCommand(conn, reader, "AUTH", l.password); err != nil {
			return fmt.Errorf("redis auth: %w", err)
		}
	}
	if l.db > 0 {
		if _, err := l.sendSimpleCommand(conn, reader, "SELECT", strconv.Itoa(l.db)); err != nil {
			return fmt.Errorf("redis select db: %w", err)
		}
	}
	return nil
}

func (l *redisRateLimiter) sendIntCommand(conn net.Conn, reader *bufio.Reader, parts ...string) (int, error) {
	if err := writeRESP(conn, parts...); err != nil {
		return 0, fmt.Errorf("write redis command: %w", err)
	}
	kind, payload, err := readRESP(reader)
	if err != nil {
		return 0, fmt.Errorf("read redis response: %w", err)
	}
	if kind == '-' {
		return 0, errors.New(payload)
	}
	if kind != ':' {
		return 0, fmt.Errorf("unexpected redis response type %q", string(kind))
	}
	value, err := strconv.Atoi(strings.TrimSpace(payload))
	if err != nil {
		return 0, fmt.Errorf("parse redis integer: %w", err)
	}
	return value, nil
}

func (l *redisRateLimiter) sendSimpleCommand(conn net.Conn, reader *bufio.Reader, parts ...string) (string, error) {
	if err := writeRESP(conn, parts...); err != nil {
		return "", fmt.Errorf("write redis command: %w", err)
	}
	kind, payload, err := readRESP(reader)
	if err != nil {
		return "", fmt.Errorf("read redis response: %w", err)
	}
	if kind == '-' {
		return "", errors.New(payload)
	}
	if kind != '+' {
		return "", fmt.Errorf("unexpected redis response type %q", string(kind))
	}
	return payload, nil
}

func writeRESP(conn net.Conn, parts ...string) error {
	if _, err := fmt.Fprintf(conn, "*%d\r\n", len(parts)); err != nil {
		return err
	}
	for _, part := range parts {
		if _, err := fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(part), part); err != nil {
			return err
		}
	}
	return nil
}

func readRESP(reader *bufio.Reader) (byte, string, error) {
	kind, err := reader.ReadByte()
	if err != nil {
		return 0, "", err
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, "", err
	}
	return kind, strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

type fallbackRateLimiter struct {
	primary  RateLimiter
	fallback RateLimiter
	logger   func(string, ...any)
}

func newFallbackRateLimiter(primary, fallback RateLimiter, logger func(string, ...any)) RateLimiter {
	if primary == nil {
		return fallback
	}
	if fallback == nil {
		return primary
	}
	return &fallbackRateLimiter{primary: primary, fallback: fallback, logger: logger}
}

func (l *fallbackRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	allowed, err := l.primary.Allow(ctx, key)
	if err == nil {
		return allowed, nil
	}
	if l.logger != nil {
		l.logger("rate limiter fallback for %s: %v", key, err)
	}
	return l.fallback.Allow(ctx, key)
}
