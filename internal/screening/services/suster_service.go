package services

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/c14220110/poliklinik-backend/internal/screening/models"
)

// SusterService menangani logika bisnis untuk suster.
type SusterService struct {
	DB *sql.DB
}

func NewSusterService(db *sql.DB) *SusterService {
	return &SusterService{DB: db}
}

// CreateSuster mendaftarkan suster baru dan meng-hash password-nya.
func (s *SusterService) CreateSuster(suster models.Suster) (int64, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(suster.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	query := `INSERT INTO Suster (Nama, Username, Password, Created_At) VALUES (?, ?, ?, ?)`
	result, err := s.DB.Exec(query, suster.Nama, suster.Username, hashedPassword, time.Now())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// AuthenticateSuster memvalidasi login suster, dengan memeriksa:
// 1. Kredensial suster (username & password)
// 2. Apakah suster memiliki shift aktif (saat ini berada di antara Jam_Mulai dan Jam_Selesai)
// 3. Apakah id_poli yang dipilih sama dengan id_poli dari shift aktif
func (s *SusterService) AuthenticateSuster(username, password string, selectedPoli int) (*models.Suster, *models.ShiftSuster, error) {
	var suster models.Suster
	query := `SELECT ID_Suster, Nama, Username, Password, Created_At FROM Suster WHERE Username = ?`
	err := s.DB.QueryRow(query, username).Scan(&suster.ID_Suster, &suster.Nama, &suster.Username, &suster.Password, &suster.CreatedAt)
	if err != nil {
		return nil, nil, err
	}
	// Periksa password menggunakan bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(suster.Password), []byte(password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	// Cek shift aktif suster: Shift_Suster di mana Jam_Mulai <= sekarang <= Jam_Selesai
	now := time.Now()
	var shift models.ShiftSuster
	queryShift := `
		SELECT ID_Shift, ID_Suster, ID_Poli, ID_Management, Jam_Mulai, Jam_Selesai 
		FROM Shift_Suster 
		WHERE ID_Suster = ? AND Jam_Mulai <= ? AND Jam_Selesai >= ?
	`
	err = s.DB.QueryRow(queryShift, suster.ID_Suster, now, now).Scan(&shift.ID_Shift, &shift.ID_Suster, &shift.ID_Poli, &shift.ID_Management, &shift.Jam_Mulai, &shift.Jam_Selesai)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.New("suster tidak memiliki shift aktif")
		}
		return nil, nil, err
	}

	// Pastikan id_poli yang dipilih sama dengan id_poli dari shift aktif
	if shift.ID_Poli != selectedPoli {
		return nil, nil, errors.New("poli yang dipilih tidak sesuai dengan shift aktif")
	}

	return &suster, &shift, nil
}
