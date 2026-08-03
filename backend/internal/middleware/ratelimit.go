// Package middleware's rate limiter is an in-memory, per-IP request cap.
// It protects against a single abusive client hammering public endpoints
// (spam submissions, login/signup brute-forcing) — it is NOT a substitute
// for infrastructure-level DDoS protection against large-scale distributed
// attacks. For that, put a CDN/WAF (e.g. Cloudflare's free tier) in front
// of the app; this middleware handles the layer that's actually within the
// application's control.
package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type visitor struct {
	count     int
	windowEnd time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
}

// NewRateLimiter allows `limit` requests per `window` duration, per client IP.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[ip]
	if !exists || now.After(v.windowEnd) {
		rl.visitors[ip] = &visitor{count: 1, windowEnd: now.Add(rl.window)}
		return true
	}
	if v.count >= rl.limit {
		return false
	}
	v.count++
	return true
}

// cleanupLoop periodically evicts expired entries so the map doesn't grow
// forever under sustained traffic from many different IPs.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			if now.After(v.windowEnd) {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware rejects requests over the limit with 429 Too Many Requests.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !rl.allow(ip) {
			writeError(w, http.StatusTooManyRequests, "too many requests — please slow down and try again shortly")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP prefers X-Forwarded-For (set by most reverse proxies/load
// balancers — Fly.io, Railway, an nginx/ALB in front of the app, etc) and
// falls back to the raw connection address for direct/local traffic.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		// X-Forwarded-For can be a comma-separated chain; the first entry
		// is the original client.
		for i, c := range fwd {
			if c == ',' {
				return fwd[:i]
			}
		}
		return fwd
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
