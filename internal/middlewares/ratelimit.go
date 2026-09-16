package middlewares

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"eglise_ujn/internal/httputil"

	"github.com/gin-gonic/gin"
)

type attemptLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func newAttemptLimiter() *attemptLimiter {
	return &attemptLimiter{attempts: make(map[string][]time.Time)}
}

func (l *attemptLimiter) allow(key string, max int, window time.Duration) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-window)
	prev := l.attempts[key]
	filtered := prev[:0]
	for _, t := range prev {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) >= max {
		l.attempts[key] = filtered
		return false
	}
	filtered = append(filtered, now)
	l.attempts[key] = filtered
	return true
}

var loginLimiter = newAttemptLimiter()

func LoginRateLimitMiddleware() gin.HandlerFunc {
	const maxAttempts = 10
	const window = 15 * time.Minute

	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		ip := strings.TrimSpace(c.ClientIP())
		email := strings.ToLower(strings.TrimSpace(c.PostForm("email")))
		key := ip + "|" + email
		if !loginLimiter.allow(key, maxAttempts, window) {
			httputil.SetFlash(c, "Trop de tentatives, réessayez dans quelques minutes")
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}
