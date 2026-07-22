package utils

import (
	"sync"
	"time"
)

// RateLimiter is a simple in-memory sliding-window rate limiter, safe for
// concurrent use. It is intended for a single-instance server: state is not
// shared across processes and resets on restart (acceptable for throttling
// abuse like OTP spam / enumeration).
type RateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

// NewRateLimiter returns a ready limiter with a background janitor that evicts
// stale keys so the map doesn't grow unbounded under many distinct keys.
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{hits: make(map[string][]time.Time)}
	go rl.janitor()
	return rl
}

// Allow records a hit for key and reports whether it is within `limit` events
// per `window`. When the limit is already reached, the hit is NOT recorded and
// false is returned.
func (rl *RateLimiter) Allow(key string, limit int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-window)
	// Compact in place, dropping timestamps older than the window.
	kept := rl.hits[key][:0]
	for _, t := range rl.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= limit {
		rl.hits[key] = kept
		return false
	}
	rl.hits[key] = append(kept, time.Now())
	return true
}

func (rl *RateLimiter) janitor() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-time.Hour)
		for key, times := range rl.hits {
			stale := true
			for _, t := range times {
				if t.After(cutoff) {
					stale = false
					break
				}
			}
			if stale {
				delete(rl.hits, key)
			}
		}
		rl.mu.Unlock()
	}
}
