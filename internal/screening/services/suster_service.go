package services

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	commonModels "github.com/c14220110/poliklinik-backend/internal/common/models"
	"github.com/c14220110/poliklinik-backend/internal/screening/models"
)

// SusterService menangani login suster.
type SusterService struct {
	DB *sql.DB
}

func NewSusterService(db *sql.DB) *SusterService {
	return &SusterService{DB: db}
}

// AuthenticateSusterUsingKaryawan memvalidasi login suster dari tabel Karyawan.
func (s *SusterService) AuthenticateSusterUsingKaryawan(username, password string, selectedPoli int) (*models.Suster, *commonModels.ShiftKaryawan, error) {
	var suster models.Suster
	queryKaryawan := "SELECT ID_Karyawan, Nama, Username, Password FROM Karyawan WHERE Username = ?"
	err := s.DB.QueryRow(queryKaryawan, username).Scan(&suster.ID_Suster, &suster.Nama, &suster.Username, &suster.Password)
	if err != nil {
		return nil, nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(suster.Password), []byte(password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}
	// Cek role untuk suster
	var roleName string
	queryRole := `
		SELECT r.Nama_Role 
		FROM Detail_Role_Karyawan drk 
		JOIN Role r ON drk.ID_Role = r.ID_Role 
		WHERE drk.ID_Karyawan = ?
		LIMIT 1
	`
	err = s.DB.QueryRow(queryRole, suster.ID_Suster).Scan(&roleName)
	if err != nil {
		return nil, nil, errors.New("failed to retrieve role")
	}
	if roleName != "Suster" {
		return nil, nil, errors.New("user is not a Suster")
	}

	// Cek shift aktif di Shift_Karyawan
	now := time.Now()
	var shift commonModels.ShiftKaryawan
	queryShift := `
		SELECT ID_Shift, ID_Karyawan, ID_Poli, Jam_Mulai, Jam_Selesai 
		FROM Shift_Karyawan 
		WHERE ID_Karyawan = ? AND Jam_Mulai <= ? AND Jam_Selesai >= ?
		LIMIT 1
	`
	err = s.DB.QueryRow(queryShift, suster.ID_Suster, now, now).Scan(&shift.ID_Shift, &shift.ID_Karyawan, &shift.ID_Poli, &shift.Jam_Mulai, &shift.Jam_Selesai)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.New("suster tidak memiliki shift aktif")
		}
		return nil, nil, err
	}
	if shift.ID_Poli != selectedPoli {
		return nil, nil, errors.New("poliklinik yang dipilih tidak sesuai dengan shift aktif")
	}

	return &suster, &shift, nil
}
