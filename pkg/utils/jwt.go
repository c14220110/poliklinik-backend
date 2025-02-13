package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims terpadu untuk semua role
type Claims struct {
	IDKaryawan string      `json:"id_karyawan"`
	Role       string      `json:"role"`
	Privileges interface{} `json:"privileges"` // bisa berupa slice atau map, sesuai kebutuhan
	IDPoli     int         `json:"id_poli,omitempty"`
	Username   string      `json:"username"`
	jwt.RegisteredClaims
}

// GenerateJWTToken membuat token JWT dengan klaim terpadu.
func GenerateJWTToken(idKaryawan string, role string, privileges interface{}, idPoli int, username string) (string, error) {
	// Ambil secret key dari environment (pastikan sudah di-set)
	jwtKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	if len(jwtKey) == 0 {
		return "", fmt.Errorf("JWT secret key is missing")
	}

	// Buat klaim
	claims := Claims{
		IDKaryawan: idKaryawan,
		Role:       role,
		Privileges: privileges,
		IDPoli:     idPoli,
		Username:   username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWTToken memvalidasi token JWT dan mengembalikan klaim terpadu.
func ValidateJWTToken(tokenString string) (*Claims, error) {
	jwtKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	if len(jwtKey) == 0 {
		return nil, fmt.Errorf("JWT secret key is missing")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Pastikan metode signing benar
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
