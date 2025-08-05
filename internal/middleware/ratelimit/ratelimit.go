package ratelimit

import (
	"sync"
	"time"
)

type UserCounter struct {
	Count     int
	LastReset time.Time
}

type RateLimiter struct {
	counters map[string]*UserCounter
	mu       sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		counters: make(map[string]*UserCounter),
	}

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *RateLimiter) IsAllowed(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	counter, exists := rl.counters[userID]

	if !exists {
		rl.counters[userID] = &UserCounter{
			Count:     1,
			LastReset: now,
		}
		return true
	}

	// Reset counter if a minute has passed
	if now.Sub(counter.LastReset) >= time.Minute {
		counter.Count = 1
		counter.LastReset = now
		return true
	}

	// Check if under limit (100 requests per minute)
	if counter.Count >= 100 {
		return false
	}

	counter.Count++
	return true
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for userID, counter := range rl.counters {
		if now.Sub(counter.LastReset) >= time.Minute {
			delete(rl.counters, userID)
		}
	}
} 