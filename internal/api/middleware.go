package api

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/nlafevers/kopds/internal/domain"
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
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// BasicAuth middleware performs HTTP Basic Authentication and stores the user in context.
func BasicAuth(userRepo domain.UserRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			unauthorized(w)
			return
		}

		user, err := userRepo.GetByUsername(r.Context(), username)
		if err != nil || user == nil {
			unauthorized(w)
			return
		}

		if !CheckPassword(user.Password, password) {
			unauthorized(w)
			return
		}

		// Store user in context
		ctx := context.WithValue(r.Context(), ContextKeyUser, username)
		GetLogger(ctx).Debug("auth success", "username", username, "auth_method", "Basic")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="KOPDS"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
