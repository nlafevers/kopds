package api

import (
	"net/http"

	"github.com/nlafevers/kopds/internal/domain"
)

// BasicAuth middleware performs HTTP Basic Authentication.
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

		next.ServeHTTP(w, r)
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="KOPDS"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
