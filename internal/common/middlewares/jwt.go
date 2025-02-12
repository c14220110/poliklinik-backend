package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/golang-jwt/jwt/v4"
)

// Definisikan tipe kustom untuk context key
type contextKey string

// Konstanta untuk key yang akan digunakan
const (
	ContextKeyUserID contextKey = "user_id"
)

func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		token, err := utils.ValidateToken(tokenStr)
		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusUnauthorized,
				"message": "Invalid token",
				"data":    nil,
			})
			return
		}

		// Lakukan type assertion ke jwt.MapClaims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["role"] != "Manajemen" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusForbidden,
				"message": "Forbidden: insufficient permissions",
				"data":    nil,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}


func JWTMiddlewareAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		token, err := utils.ValidateToken(tokenStr)
		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusUnauthorized,
				"message": "Invalid token",
				"data":    nil,
			})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["role"] != "Administrasi" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusForbidden,
				"message": "Forbidden: insufficient permissions",
				"data":    nil,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// JWTMiddlewareManagement memastikan hanya pengguna dengan role "Manajemen" yang dapat mengakses endpoint.
func JWTMiddlewareManagement(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		token, err := utils.ValidateToken(tokenStr)
		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusUnauthorized,
				"message": "Invalid token",
				"data":    nil,
			})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["role"] != "Manajemen" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusForbidden,
				"message": "Forbidden: insufficient permissions",
				"data":    nil,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func JWTMiddlewareSuster(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		token, err := utils.ValidateToken(tokenStr)
		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusUnauthorized,
				"message": "Invalid token",
				"data":    nil,
			})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusUnauthorized,
				"message": "Invalid token claims",
				"data":    nil,
			})
			return
		}
		// Pastikan role adalah "Suster"
		if role, ok := claims["role"].(string); !ok || role != "Suster" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusForbidden,
				"message": "Forbidden: insufficient permissions",
				"data":    nil,
			})
			return
		}
		// Ekstrak "user_id"
		var id int
		if userID, exists := claims["user_id"]; exists {
			switch v := userID.(type) {
			case float64:
				id = int(v)
			case int:
				id = v
			case string:
				parsed, err := strconv.Atoi(v)
				if err != nil {
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"status":  http.StatusUnauthorized,
						"message": "Invalid user_id format in token",
						"data":    nil,
					})
					return
				}
				id = parsed
			default:
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  http.StatusUnauthorized,
					"message": "Invalid user_id type in token",
					"data":    nil,
				})
				return
			}
			if id <= 0 {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  http.StatusUnauthorized,
					"message": "Invalid user_id value in token",
					"data":    nil,
				})
				return
			}
			// Simpan user_id ke context dengan key tipe kustom
			ctx := context.WithValue(r.Context(), ContextKeyUserID, id)
			r = r.WithContext(ctx)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusUnauthorized,
				"message": "Invalid or missing operator ID in token",
				"data":    nil,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}