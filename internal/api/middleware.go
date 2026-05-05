package api

import (
	"net/http"
)

// BasicAuth middleware performs HTTP Basic Authentication.
func (h *Handler) BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			h.unauthorized(w)
			return
		}

		user, err := h.UserRepo.GetByUsername(r.Context(), username)
		if err != nil || user == nil {
			h.unauthorized(w)
			return
		}

		if !CheckPasswordHash(password, user.Password) {
			h.unauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="KOPDS"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
