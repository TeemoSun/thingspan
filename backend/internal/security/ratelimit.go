package security

import (
	"sync"
	"time"
)

const (
	LoginRateLimit  = 5
	LoginRateWindow = 60 * time.Second
)

type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		attempts: make(map[string][]time.Time),
	}
}

// CheckLimit returns true if allowed, false if limit exceeded
func (rl *RateLimiter) CheckLimit(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-LoginRateWindow)

	times := rl.attempts[ip]
	// filter old attempts
	validIndex := 0
	for i, t := range times {
		if t.After(windowStart) {
			validIndex = i
			break
		}
		if i == len(times)-1 {
			validIndex = len(times)
		}
	}
	times = times[validIndex:]
	rl.attempts[ip] = times

	return len(times) < LoginRateLimit
}

// RecordFailure records a failed login attempt for the given IP
func (rl *RateLimiter) RecordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-LoginRateWindow)

	times := rl.attempts[ip]
	validIndex := 0
	for i, t := range times {
		if t.After(windowStart) {
			validIndex = i
			break
		}
		if i == len(times)-1 {
			validIndex = len(times)
		}
	}
	times = append(times[validIndex:], now)
	rl.attempts[ip] = times

	// periodic cleanup if map grows large
	if len(rl.attempts) > 1024 {
		for k, v := range rl.attempts {
			if len(v) == 0 || !v[len(v)-1].After(windowStart) {
				delete(rl.attempts, k)
			}
		}
	}
}

