package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/codetheuri/tusk/pkg/logger"
)

type clientVisitor struct {
	tokens     float64
	lastSeen   time.Time
	maxTokens  float64
	refillRate float64 // tokens per second
}

// RateLimiter provides IP-based token bucket rate limiting for endpoints.
type RateLimiter struct {
	mu         sync.Mutex
	visitors   map[string]*clientVisitor
	maxTokens  float64
	refillRate float64
	log        logger.Logger
}

// NewRateLimiter creates a RateLimiter with specified max tokens (capacity) and refill rate per second.
func NewRateLimiter(maxTokens float64, refillRate float64, log logger.Logger) *RateLimiter {
	rl := &RateLimiter{
		visitors:   make(map[string]*clientVisitor),
		maxTokens:  maxTokens,
		refillRate: refillRate,
		log:        log,
	}

	// Periodically clean up stale visitors (older than 10 minutes)
	go func() {
		for {
			time.Sleep(3 * time.Minute)
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > 10*time.Minute {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

// Limit returns a middleware that limits incoming HTTP requests per remote IP.
func (rl *RateLimiter) Limit() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if forwardFor := r.Header.Get("X-Forwarded-For"); forwardFor != "" {
				ip = forwardFor
			}

			rl.mu.Lock()
			v, exists := rl.visitors[ip]
			now := time.Now()

			if !exists {
				v = &clientVisitor{
					tokens:     rl.maxTokens - 1,
					lastSeen:   now,
					maxTokens:  rl.maxTokens,
					refillRate: rl.refillRate,
				}
				rl.visitors[ip] = v
				rl.mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			// Refill tokens based on elapsed time
			elapsed := now.Sub(v.lastSeen).Seconds()
			v.tokens += elapsed * v.refillRate
			if v.tokens > v.maxTokens {
				v.tokens = v.maxTokens
			}
			v.lastSeen = now

			if v.tokens < 1 {
				rl.mu.Unlock()
				rl.log.Warn("Rate limit exceeded", "ip", ip, "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"success":false,"message":"Too many requests. Please try again later."}`))
				return
			}

			v.tokens -= 1
			rl.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}
