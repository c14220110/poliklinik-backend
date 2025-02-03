package utils

import (
	"errors"
	"time"

	"github.com/c14220110/poliklinik-backend/config"
	"github.com/golang-jwt/jwt/v4"
)

var jwtSecret []byte

// Inisialisasi secret JWT dari konfigurasi.
func init() {
	cfg := config.LoadConfig()
	jwtSecret = []byte(cfg.JWTSecret)
}

// GenerateToken membuat JWT token dengan klaim dasar.
func GenerateToken(userID int, username string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(72 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken memvalidasi token JWT dan mengembalikan token jika valid.
func ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Pastikan metode signing adalah HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	return token, err
}
