package api

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlafevers/kopds/internal/domain"
)

func TestGenerateRequestID(t *testing.T) {
	t.Run("returns 32-char hex string", func(t *testing.T) {
		id := generateRequestID()
		if len(id) != 32 {
			t.Errorf("expected 32 hex chars, got %d: %q", len(id), id)
		}
		if _, err := hex.DecodeString(id); err != nil {
			t.Errorf("expected valid hex string, got %q: %v", id, err)
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		id1 := generateRequestID()
		id2 := generateRequestID()
		if id1 == id2 {
			t.Error("expected unique request IDs, got identical values")
		}
	})

	t.Run("request_id present in log context", func(t *testing.T) {
		handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Context().Value(ContextKeyRequestID)
			if id == nil {
				t.Error("expected request_id in context")
				return
			}
			idStr, ok := id.(string)
			if !ok || len(idStr) != 32 {
				t.Errorf("expected 32-char hex request_id, got %q", idStr)
			}
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	})
}

func TestBasicAuth(t *testing.T) {
	password := "secret"
	hash, _ := HashPassword(password)

	userRepo := &mockUserRepo{getByUsernameFunc: func(ctx context.Context, username string) (*domain.User, error) {
		if username == "admin" {
			return &domain.User{Username: "admin", Password: hash}, nil
		}
		return nil, nil
	},
	}

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	authMiddleware := BasicAuth(userRepo, nextHandler)
	tests := []struct {
		name           string
		username       string
		password       string
		expectedStatus int
	}{
		{"Valid Credentials", "admin", "secret", http.StatusOK},
		{"Invalid Password", "admin", "wrong", http.StatusUnauthorized},
		{"Invalid Username", "unknown", "secret", http.StatusUnauthorized},
		{"No Credentials", "", "", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			if tt.username != "" {
				req.SetBasicAuth(tt.username, tt.password)
			}

			rr := httptest.NewRecorder()
			authMiddleware.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusUnauthorized {
				if rr.Header().Get("WWW-Authenticate") == "" {
					t.Error("expected WWW-Authenticate header")
				}
			}
		})
	}
}
