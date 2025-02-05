package services

import (
	"database/sql"
	"errors"

	//"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/c14220110/poliklinik-backend/internal/dokter/models"
)

type DokterService struct {
	DB *sql.DB
}

func NewDokterService(db *sql.DB) *DokterService {
	return &DokterService{DB: db}
}

// CreateDokter mendaftarkan dokter baru dengan meng-hash password-nya.
func (s *DokterService) CreateDokter(d models.Dokter) (int64, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(d.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	query := `INSERT INTO Dokter (Nama, Username, Password, Spesialisasi) VALUES (?, ?, ?, ?)`
	result, err := s.DB.Exec(query, d.Nama, d.Username, hashedPassword, d.Spesialisasi)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// AuthenticateDokter memvalidasi login dokter.
func (s *DokterService) AuthenticateDokter(username, password string) (*models.Dokter, error) {
	var dokter models.Dokter
	query := `SELECT ID_Dokter, Nama, Username, Password, Spesialisasi FROM Dokter WHERE Username = ?`
	err := s.DB.QueryRow(query, username).Scan(&dokter.ID_Dokter, &dokter.Nama, &dokter.Username, &dokter.Password, &dokter.Spesialisasi)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(dokter.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	return &dokter, nil
}
