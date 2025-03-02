package middlewares

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

// RequirePrivilege memeriksa apakah klaim JWT memiliki privilege yang dibutuhkan.
func RequirePrivilege(requiredPriv int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rawClaims := c.Get(string(ContextKeyClaims))
			if rawClaims == nil {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"status":  http.StatusUnauthorized,
					"message": "Missing or invalid JWT claims",
					"data":    nil,
				})
			}
			claims, ok := rawClaims.(map[string]interface{})
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"status":  http.StatusUnauthorized,
					"message": "Invalid JWT claims format",
					"data":    nil,
				})
			}

			// Cek key "privileges" dalam klaim
			var privSlice []interface{}
			if privVal, exists := claims["privileges"]; exists {
				switch v := privVal.(type) {
				case []interface{}:
					privSlice = v
				case map[string]interface{}:
					if arr, exists := v["privileges"]; exists {
						if arrSlice, ok := arr.([]interface{}); ok {
							privSlice = arrSlice
						}
					}
				}
			}
			if privSlice == nil {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"status":  http.StatusForbidden,
					"message": "User does not have privileges",
					"data":    nil,
				})
			}

			found := false
			for _, item := range privSlice {
				switch val := item.(type) {
				case float64:
					if int(val) == requiredPriv {
						found = true
						break
					}
				case string:
					if n, err := strconv.Atoi(val); err == nil && n == requiredPriv {
						found = true
						break
					}
				}
			}

			if !found {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"status":  http.StatusForbidden,
					"message": "Anda tidak memiliki hak akses",
					"data":    nil,
				})
			}

			return next(c)
		}
	}
}
