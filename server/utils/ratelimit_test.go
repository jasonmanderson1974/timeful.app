package utils

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiterAllowsUpToLimitThenDenies(t *testing.T) {
	rl := NewRateLimiter()
	for i := 1; i <= 3; i++ {
		if !rl.Allow("k", 3, time.Minute) {
			t.Fatalf("hit %d should be allowed (limit 3)", i)
		}
	}
	if rl.Allow("k", 3, time.Minute) {
		t.Fatalf("4th hit should be denied (limit 3)")
	}
}

func TestRateLimiterKeysAreIndependent(t *testing.T) {
	rl := NewRateLimiter()
	if !rl.Allow("a", 1, time.Minute) {
		t.Fatalf("first hit for a should be allowed")
	}
	if !rl.Allow("b", 1, time.Minute) {
		t.Fatalf("first hit for b should be allowed (independent key)")
	}
	if rl.Allow("a", 1, time.Minute) {
		t.Fatalf("second hit for a should be denied")
	}
}

func TestRateLimiterWindowExpiry(t *testing.T) {
	rl := NewRateLimiter()
	window := 40 * time.Millisecond
	if !rl.Allow("k", 1, window) {
		t.Fatalf("first hit should be allowed")
	}
	if rl.Allow("k", 1, window) {
		t.Fatalf("second hit within window should be denied")
	}
	time.Sleep(window + 20*time.Millisecond)
	if !rl.Allow("k", 1, window) {
		t.Fatalf("hit after the window elapsed should be allowed again")
	}
}

func TestRateLimiterDeniedHitsDoNotExtendWindow(t *testing.T) {
	// A denied hit must not be recorded, otherwise repeated attempts would keep
	// pushing the window forward and never recover.
	rl := NewRateLimiter()
	window := 40 * time.Millisecond
	rl.Allow("k", 1, window) // consume the single slot
	rl.Allow("k", 1, window) // denied — must not be recorded
	rl.Allow("k", 1, window) // denied — must not be recorded
	time.Sleep(window + 20*time.Millisecond)
	if !rl.Allow("k", 1, window) {
		t.Fatalf("denied hits should not extend the window; should be allowed after expiry")
	}
}

func TestRateLimiterConcurrentSafe(t *testing.T) {
	// Race-detector smoke test: many goroutines hammering the same key must not
	// exceed the limit and must not race.
	rl := NewRateLimiter()
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow("shared", 10, time.Minute) {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != 10 {
		t.Fatalf("expected exactly 10 allowed out of 100 concurrent, got %d", allowed)
	}
}
