package services

import (
	"database/sql"
	"errors"
	"time"

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

// AuthenticateDokter memvalidasi login dokter dengan cek shift aktif.
// Param: username, password, dan selectedPoli (id_poli yang dipilih oleh dokter dari dropdown).
// Jika dokter memiliki shift aktif dan selectedPoli sama dengan shift aktif, kembalikan data dokter dan shift.
func (s *DokterService) AuthenticateDokter(username, password string, selectedPoli int) (*models.Dokter, *models.ShiftDokter, error) {
	var dokter models.Dokter
	queryDokter := `SELECT ID_Dokter, Nama, Username, Password, Spesialisasi FROM Dokter WHERE Username = ?`
	err := s.DB.QueryRow(queryDokter, username).Scan(&dokter.ID_Dokter, &dokter.Nama, &dokter.Username, &dokter.Password, &dokter.Spesialisasi)
	if err != nil {
		return nil, nil, err
	}

	// Verifikasi password menggunakan bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(dokter.Password), []byte(password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	// Cek shift aktif dokter: Cari record di Shift_Dokter di mana
	// Jam_Mulai <= now <= Jam_Selesai dan ID_Dokter = dokter.ID_Dokter.
	now := time.Now()
	var shift models.ShiftDokter
	queryShift := `
		SELECT ID_Shift, ID_Dokter, ID_Poli, ID_Management, Jam_Mulai, Jam_Selesai 
		FROM Shift_Dokter 
		WHERE ID_Dokter = ? AND Jam_Mulai <= ? AND Jam_Selesai >= ?
	`
	err = s.DB.QueryRow(queryShift, dokter.ID_Dokter, now, now).Scan(&shift.ID_Shift, &shift.ID_Dokter, &shift.ID_Poli, &shift.ID_Management, &shift.Jam_Mulai, &shift.Jam_Selesai)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.New("dokter tidak memiliki shift aktif")
		}
		return nil, nil, err
	}

	// Pastikan id_poli yang dipilih (selectedPoli) sama dengan id_poli dari shift aktif.
	if shift.ID_Poli != selectedPoli {
		return nil, nil, errors.New("poliklinik yang dipilih tidak sesuai dengan shift aktif")
	}

	return &dokter, &shift, nil
}
