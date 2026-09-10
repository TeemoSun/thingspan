package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"thingspan/internal/models"
	"thingspan/internal/security"
)

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func WriteError(w http.ResponseWriter, status int, detail string) {
	WriteJSON(w, status, models.ErrorDetail{Detail: detail})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func RequireAuth(jwtm *security.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				WriteError(w, http.StatusUnauthorized, "未登录")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				WriteError(w, http.StatusUnauthorized, "未登录")
				return
			}

			tokenStr := parts[1]
			_, err := jwtm.DecodeToken(tokenStr, "access")
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "登录已过期")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

