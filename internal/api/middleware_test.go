package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nlafevers/kopds/internal/domain"
	"github.com/nlafevers/kopds/pkg/utils"
)

func TestBasicAuth(t *testing.T) {
	password := "secret"
	hash, _ := HashPassword(password)

	linkGen := utils.NewLinkGenerator("http://localhost:8080")
	userRepo := &mockUserRepo{
		getByUsernameFunc: func(ctx context.Context, username string) (*domain.User, error) {
			if username == "admin" {
				return &domain.User{Username: "admin", Password: hash}, nil
			}
			return nil, nil
		},
	}

	h := NewHandler(nil, userRepo, linkGen, nil, "")

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	authMiddleware := h.BasicAuth(nextHandler)

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
