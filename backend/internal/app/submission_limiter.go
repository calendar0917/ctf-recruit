package app

import (
	"sync"
	"time"
)

type submissionLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	window  time.Duration
	max     int
	buckets map[string]submissionBucket
}

type submissionBucket struct {
	count     int
	windowEnd time.Time
}

func newSubmissionLimiter(window time.Duration, max int) *submissionLimiter {
	return &submissionLimiter{
		now:     time.Now,
		window:  window,
		max:     max,
		buckets: make(map[string]submissionBucket),
	}
}

func (l *submissionLimiter) Allow(key string) bool {
	if l == nil || l.window <= 0 || l.max <= 0 {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().UTC()
	bucket, ok := l.buckets[key]
	if !ok || !now.Before(bucket.windowEnd) {
		l.buckets[key] = submissionBucket{count: 1, windowEnd: now.Add(l.window)}
		return true
	}
	if bucket.count >= l.max {
		return false
	}
	bucket.count++
	l.buckets[key] = bucket
	return true
}
