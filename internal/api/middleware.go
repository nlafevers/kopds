package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nlafevers/kopds/internal/domain"
	"golang.org/x/time/rate"
)

type responseWriter struct {
	http.ResponseWriter
	status    int
	size      int64
	errorBody []byte
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	if rw.status >= 500 {
		rw.errorBody = append(rw.errorBody, b...)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += int64(n)
	return n, err
}

// LoggingMiddleware logs HTTP requests and responses.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		requestID := generateRequestID()
		logger := slog.With(
			"method", r.Method,
			"path", r.URL.Path,
			"request_id", requestID,
		)

		ctx := context.WithValue(r.Context(), ContextKeyRequestID, requestID)
		ctx = context.WithValue(ctx, ContextKeyLogger, logger)
		r = r.WithContext(ctx)

		rw := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		user := GetUser(r.Context())

		fields := []any{
			"status_code", rw.status,
			"duration", duration,
			"remote_addr", r.RemoteAddr,
		}
		if user != "" {
			fields = append(fields, "user", user)
		}

		if rw.status >= 500 {
			fields = append(fields, "error_detail", string(rw.errorBody))
			logger.Error("server error", fields...)
		} else if rw.status >= 400 {
			logger.Warn("client error", fields...)
		} else {
			logger.Info("request completed", fields...)
		}

		logger.Debug("response diagnostics",
			"status_code", rw.status,
			"duration", duration,
			"response_size", rw.size,
			"user", user,
		)
	})
}

func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// BasicAuth middleware performs HTTP Basic Authentication and stores the user in context.
// When a limiter is provided, failed auth attempts are counted against it; if the limiter
// denies the request, 429 Too Many Requests is returned instead of 401 Unauthorized.
func BasicAuth(userRepo domain.UserRepository, limiter *IPRateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			rateLimitedUnauthorized(w, r, limiter)
			return
		}

		user, err := userRepo.GetByUsername(r.Context(), username)
		if err != nil || user == nil {
			rateLimitedUnauthorized(w, r, limiter)
			return
		}

		if !CheckPassword(user.Password, password) {
			rateLimitedUnauthorized(w, r, limiter)
			return
		}

		// Store user in context
		ctx := context.WithValue(r.Context(), ContextKeyUser, username)
		GetLogger(ctx).Debug("auth success", "username", username, "auth_method", "Basic")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// rateLimitedUnauthorized checks the rate limiter on a failed auth attempt.
// If the limiter is non-nil and the IP has exceeded its limit, 429 is returned.
// Otherwise 401 Unauthorized is returned.
func rateLimitedUnauthorized(w http.ResponseWriter, r *http.Request, limiter *IPRateLimiter) {
	if limiter != nil {
		ip := clientIP(r, limiter.trustProxy)
		if !limiter.GetLimiter(ip).Allow() {
			GetLogger(r.Context()).Warn("rate limit exceeded on failed auth", "ip", ip, "source", "API")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
	}
	unauthorized(w)
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="KOPDS"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}

// IPRateLimiter handles rate limiting per IP address.
type IPRateLimiter struct {
	ips        map[string]*rate.Limiter
	mu         sync.RWMutex
	r          rate.Limit
	b          int
	trustProxy bool
}

func NewIPRateLimiter(r rate.Limit, b int, trustProxy bool) *IPRateLimiter {
	return &IPRateLimiter{
		ips:        make(map[string]*rate.Limiter),
		r:          r,
		b:          b,
		trustProxy: trustProxy,
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.RLock()
	limiter, exists := i.ips[ip]
	i.mu.RUnlock()

	if !exists {
		i.mu.Lock()
		// Double-check after acquiring write lock.
		if limiter, exists = i.ips[ip]; !exists {
			limiter = rate.NewLimiter(i.r, i.b)
			i.ips[ip] = limiter
		}
		i.mu.Unlock()
	}

	return limiter
}

// clientIP returns the client's IP address. If trustProxy is true and the
// X-Forwarded-For header is present, the first (leftmost) address is used.
// Otherwise the IP is taken from r.RemoteAddr.
func clientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ip := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
			if ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimitMiddleware applies rate limiting per IP.
func RateLimitMiddleware(limiter *IPRateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r, limiter.trustProxy)
		if !limiter.GetLimiter(ip).Allow() {
			GetLogger(r.Context()).Warn("rate limit exceeded", "ip", ip, "source", "API")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
