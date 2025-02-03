package services

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
)

// AdministrasiService menangani logika autentikasi dan pendaftaran administrasi.
type AdministrasiService struct {
	DB *sql.DB
}

func NewAdministrasiService(db *sql.DB) *AdministrasiService {
	return &AdministrasiService{DB: db}
}

// Authenticate memeriksa kredensial user administrasi.
// Untuk produksi, kami menggunakan bcrypt untuk membandingkan password.
func (s *AdministrasiService) Authenticate(username, password string) (*models.Administrasi, error) {
	var admin models.Administrasi
	query := "SELECT ID_Admin, Nama, Username, Password, Created_At FROM Administrasi WHERE Username = ?"
	err := s.DB.QueryRow(query, username).Scan(&admin.ID, &admin.Nama, &admin.Username, &admin.Password, &admin.CreatedAt)
	if err != nil {
		return nil, err
	}
	// Bandingkan hash yang ada dengan password yang dikirimkan
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	return &admin, nil
}

// CreateAdmin membuat admin baru dengan menghash password terlebih dahulu.
func (s *AdministrasiService) CreateAdmin(admin models.Administrasi) (int64, error) {
	// Hash password dengan cost default
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(admin.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	query := "INSERT INTO Administrasi (Nama, Username, Password, Created_At) VALUES (?, ?, ?, ?)"
	result, err := s.DB.Exec(query, admin.Nama, admin.Username, hashedPassword, time.Now())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
