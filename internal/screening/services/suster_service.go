package services

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	commonModels "github.com/c14220110/poliklinik-backend/internal/common/models"
	"github.com/c14220110/poliklinik-backend/internal/screening/models"
)

type SusterService struct {
	DB *sql.DB
}

func NewSusterService(db *sql.DB) *SusterService {
	return &SusterService{DB: db}
}

// AuthenticateSusterUsingKaryawan memvalidasi login suster, mengambil role, privileges,
// dan memeriksa apakah shift aktif berdasarkan custom_jam_mulai dan custom_jam_selesai.
func (s *SusterService) AuthenticateSusterUsingKaryawan(username, password string, selectedPoli int) (*models.Suster, *commonModels.ShiftKaryawan, error) {
	var suster models.Suster
	// Ambil data karyawan
	queryKaryawan := "SELECT ID_Karyawan, Nama, Username, Password FROM Karyawan WHERE Username = ?"
	err := s.DB.QueryRow(queryKaryawan, username).Scan(&suster.ID_Suster, &suster.Nama, &suster.Username, &suster.Password)
	if err != nil {
		return nil, nil, err
	}
	// Verifikasi password
	if err := bcrypt.CompareHashAndPassword([]byte(suster.Password), []byte(password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}
	// Ambil role dan role ID
	var roleName string
	var roleID int
	queryRole := `
		SELECT r.Nama_Role, drk.ID_Role
		FROM Detail_Role_Karyawan drk 
		JOIN Role r ON drk.ID_Role = r.ID_Role 
		WHERE drk.ID_Karyawan = ?
		LIMIT 1
	`
	err = s.DB.QueryRow(queryRole, suster.ID_Suster).Scan(&roleName, &roleID)
	if err != nil {
		return nil, nil, errors.New("failed to retrieve role")
	}
	if roleName != "Suster" {
		return nil, nil, errors.New("user is not a Suster")
	}
	suster.ID_Role = roleID

	// Ambil daftar privilege
	rows, err := s.DB.Query("SELECT id_privilege FROM Detail_Privilege_Karyawan WHERE id_karyawan = ?", suster.ID_Suster)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var privileges []int
	for rows.Next() {
		var priv int
		if err := rows.Scan(&priv); err != nil {
			return nil, nil, err
		}
		privileges = append(privileges, priv)
	}
	suster.Privileges = privileges

	// Cek shift aktif berdasarkan custom_jam_mulai dan custom_jam_selesai
	var shift commonModels.ShiftKaryawan
	// Query untuk shift aktif pada poli yang dipilih dan hari ini
	queryActive := `
		SELECT sk.ID_Shift_Karyawan, sk.custom_jam_mulai, sk.custom_jam_selesai, sk.ID_Poli
		FROM Shift_Karyawan sk
		WHERE sk.ID_Karyawan = ?
		  AND sk.ID_Poli = ?
		  AND sk.Tanggal = CURDATE()
		  AND (
		      (sk.custom_jam_mulai < sk.custom_jam_selesai AND CURTIME() BETWEEN sk.custom_jam_mulai AND sk.custom_jam_selesai)
		      OR (sk.custom_jam_mulai > sk.custom_jam_selesai AND (CURTIME() >= sk.custom_jam_mulai OR CURTIME() < sk.custom_jam_selesai))
		  )
		LIMIT 1
	`
	err = s.DB.QueryRow(queryActive, suster.ID_Suster, selectedPoli).Scan(&shift.ID_Shift_Karyawan, &shift.CustomJamMulai, &shift.CustomJamSelesai, &shift.ID_Poli)
	if err == nil {
		// Shift aktif ditemukan
		return &suster, &shift, nil
	}

	// Tidak ada shift aktif; periksa apakah ada shift untuk poli tersebut hari ini
	var shiftRecord commonModels.ShiftKaryawan
	queryAny := `
		SELECT sk.ID_Shift_Karyawan, sk.custom_jam_mulai, sk.custom_jam_selesai, sk.ID_Poli
		FROM Shift_Karyawan sk
		WHERE sk.ID_Karyawan = ?
		  AND sk.ID_Poli = ?
		  AND sk.Tanggal = CURDATE()
		LIMIT 1
	`
	err = s.DB.QueryRow(queryAny, suster.ID_Suster, selectedPoli).Scan(&shiftRecord.ID_Shift_Karyawan, &shiftRecord.CustomJamMulai, &shiftRecord.CustomJamSelesai, &shiftRecord.ID_Poli)
	if err == sql.ErrNoRows {
		// Tidak ada shift di poli tersebut hari ini; periksa shift di poli lain
		var otherPoli int
		queryOther := `
			SELECT sk.ID_Poli
			FROM Shift_Karyawan sk
			WHERE sk.ID_Karyawan = ? AND sk.Tanggal = CURDATE()
			LIMIT 1
		`
		err = s.DB.QueryRow(queryOther, suster.ID_Suster).Scan(&otherPoli)
		if err == nil {
			return nil, nil, errors.New("shift aktif di poli lain")
		}
		return nil, nil, errors.New("tidak ada shift aktif hari ini untuk poli yang dipilih")
	}

	// Ada shift, tetapi saat ini tidak aktif. Lakukan pengecekan waktu.
	currentTimeStr := time.Now().Format("15:04:05")
	parsedCurrent, _ := time.Parse("15:04:05", currentTimeStr)
	parsedMulai, err1 := time.Parse("15:04:05", shiftRecord.CustomJamMulai)
	parsedSelesai, err2 := time.Parse("15:04:05", shiftRecord.CustomJamSelesai)
	if err1 != nil || err2 != nil {
		return nil, nil, errors.New("format waktu shift tidak valid")
	}

	if parsedCurrent.Before(parsedMulai) {
		return nil, nil, errors.New("shift akan aktif nanti pada pukul " + shiftRecord.CustomJamMulai)
	} else if parsedCurrent.After(parsedSelesai) {
		return nil, nil, errors.New("shift sudah berakhir")
	}
	// Default fallback, meskipun seharusnya sudah tercakup
	return nil, nil, errors.New("shift tidak aktif saat ini")
}
