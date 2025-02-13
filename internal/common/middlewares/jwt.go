package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/c14220110/poliklinik-backend/pkg/utils"
)

// Buat tipe kustom untuk context key
type contextKey string

const (
  ContextKeyUserID contextKey = "user_id"
)

// JWTMiddleware adalah middleware terpadu untuk memvalidasi token JWT.
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ambil header Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusUnauthorized,
				"message": "Authorization header missing",
				"data":    nil,
			})
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusUnauthorized,
				"message": "Invalid authorization header",
				"data":    nil,
			})
			return
		}
		tokenStr := parts[1]
		claims, err := utils.ValidateJWTToken(tokenStr)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusUnauthorized,
				"message": "Invalid token: " + err.Error(),
				"data":    nil,
			})
			return
		}

		// Simpan claims ke context (claims sudah bertipe *utils.Claims)
		ctx := context.WithValue(r.Context(), ContextKeyUserID, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
