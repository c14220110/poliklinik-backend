package middlewares

import (
	"net/http"
	"strings"

	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Definisikan tipe kustom untuk context key
type contextKey string

const (
    ContextKeyClaims contextKey = "claims"
)

// JWTMiddleware untuk autentikasi JWT dengan Echo
func JWTMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Ambil header Authorization
            authHeader := c.Request().Header.Get("Authorization")
            if authHeader == "" {
                return c.JSON(http.StatusUnauthorized, map[string]interface{}{
                    "status":  http.StatusUnauthorized,
                    "message": "Authorization header missing",
                    "data":    nil,
                })
            }

            parts := strings.Split(authHeader, " ")
            if len(parts) != 2 || parts[0] != "Bearer" {
                return c.JSON(http.StatusUnauthorized, map[string]interface{}{
                    "status":  http.StatusUnauthorized,
                    "message": "Invalid authorization header",
                    "data":    nil,
                })
            }

            tokenStr := parts[1]
            claims, err := utils.ValidateJWTToken(tokenStr)
            if err != nil {
                return c.JSON(http.StatusUnauthorized, map[string]interface{}{
                    "status":  http.StatusUnauthorized,
                    "message": "Invalid token: " + err.Error(),
                    "data":    nil,
                })
            }

            // Simpan claims ke context Echo
            c.Set(string(ContextKeyClaims), claims)
            return next(c)
        }
    }
}

// CORSMiddleware untuk mengatur CORS
func CORSMiddleware() echo.MiddlewareFunc {
    return middleware.CORSWithConfig(middleware.CORSConfig{
        AllowOrigins: []string{"http://localhost:3000"}, // Ganti dengan domain frontend spesifik
        AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
        AllowHeaders: []string{
            echo.HeaderOrigin,
            echo.HeaderContentType,
            echo.HeaderAccept,
            "Authorization", // Pastikan Authorization diizinkan
        },
    })
}