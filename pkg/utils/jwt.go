package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims terpadu dengan field flat untuk id_role dan privileges.
type Claims struct {
	IDKaryawan string   `json:"id_karyawan"`
	Role       string   `json:"role"`
	IDRole     int      `json:"id_role"`
	Privileges []int    `json:"privileges"`
	IDPoli     int      `json:"id_poli,omitempty"`
	Username   string   `json:"username"`
	jwt.RegisteredClaims
}

// GenerateJWTToken membuat token JWT dengan payload flat dan exp sesuai parameter.
func GenerateJWTToken(idKaryawan string, role string, idRole int, privileges []int, idPoli int, username string, exp time.Time) (string, error) {
	jwtKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	if len(jwtKey) == 0 {
		return "", fmt.Errorf("JWT secret key is missing")
	}

	claims := Claims{
		IDKaryawan: idKaryawan,
		Role:       role,
		IDRole:     idRole,
		Privileges: privileges,
		IDPoli:     idPoli,
		Username:   username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
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
