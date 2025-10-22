package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter stores rate limiters for each IP
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
	requests int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		requests: requests,
		window:   window,
	}
}

// GetLimiter returns a rate limiter for the given IP
func (rl *RateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[ip]
	if !exists {
		// Calculate rate: requests per second
		ratePerSecond := float64(rl.requests) / rl.window.Seconds()
		limiter = rate.NewLimiter(rate.Limit(ratePerSecond), rl.requests)
		rl.limiters[ip] = limiter
	}

	return limiter
}

// Middleware returns the rate limiting middleware
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			ip := r.RemoteAddr
			// Handle X-Forwarded-For header for proxied requests
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				ip = forwarded
			}

			limiter := rl.GetLimiter(ip)
			if !limiter.Allow() {
				writeError(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CleanupOldLimiters removes limiters that haven't been used recently
func (rl *RateLimiter) CleanupOldLimiters() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			// In production, you'd track last access time and remove old entries
			// For now, we'll just clear all to prevent memory leaks
			rl.limiters = make(map[string]*rate.Limiter)
			rl.mu.Unlock()
		}
	}()
}
