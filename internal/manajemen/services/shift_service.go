package services

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ShiftService struct {
	DB *sql.DB
}

func NewShiftService(db *sql.DB) *ShiftService {
	return &ShiftService{DB: db}
}

func (s *ShiftService) AssignShift(idPoli, idKaryawan, idRole, idShift, idManagement int, tanggalStr string) (int64, error) {
	// 0. Cek apakah karyawan memiliki role sesuai parameter (dilakukan di luar transaksi).
	var roleCount int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM Detail_Role_Karyawan WHERE id_karyawan = ? AND id_role = ?", idKaryawan, idRole).Scan(&roleCount)
	if err != nil {
		return 0, fmt.Errorf("failed to check role for karyawan: %v", err)
	}
	if roleCount == 0 {
		return 0, fmt.Errorf("karyawan dengan id %d tidak memiliki role %d", idKaryawan, idRole)
	}

	// 1. Validasi format tanggal
	_, err = time.Parse("2006-01-02", tanggalStr)
	if err != nil {
		return 0, fmt.Errorf("format tanggal tidak valid: %v", err)
	}

	// Mulai transaksi
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}

	// 2. Cek apakah karyawan sudah memiliki shift yang sama di poli pada tanggal ini
	var existingCount int
	err = tx.QueryRow(
		"SELECT COUNT(*) FROM Shift_Karyawan WHERE id_karyawan = ? AND id_poli = ? AND id_shift = ? AND tanggal = ?",
		idKaryawan, idPoli, idShift, tanggalStr,
	).Scan(&existingCount)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to check existing shift: %v", err)
	}
	if existingCount > 0 {
		tx.Rollback()
		return 0, fmt.Errorf("User dengan role %d sudah memiliki shift %d di poli %d pada tanggal %s", idRole, idShift, idPoli, tanggalStr)
	}

	// 3. Ambil data Shift untuk mendapatkan jam_mulai dan jam_selesai default
	var jamMulai, jamSelesai string
	queryShift := "SELECT jam_mulai, jam_selesai FROM Shift WHERE id_shift = ?"
	err = tx.QueryRow(queryShift, idShift).Scan(&jamMulai, &jamSelesai)
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("id_shift %d tidak ditemukan", idShift)
		}
		return 0, err
	}

	// 4. Insert ke tabel Shift_Karyawan
	insertQuery := `
		INSERT INTO Shift_Karyawan (id_poli, id_shift, id_karyawan, custom_jam_mulai, custom_jam_selesai, tanggal)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	res, err := tx.Exec(insertQuery, idPoli, idShift, idKaryawan, jamMulai, jamSelesai, tanggalStr)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	idShiftKaryawan, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// 5. Insert ke tabel Management_Shift_Karyawan
	insertManagementShiftQuery := `
		INSERT INTO Management_Shift_Karyawan (id_management, id_shift_karyawan, created_by, updated_by, deleted_by)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(insertManagementShiftQuery, idManagement, idShiftKaryawan, idManagement, idManagement, 0)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Commit transaksi
	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return idShiftKaryawan, nil
}


func (s *ShiftService) UpdateCustomShift(idShiftKaryawan int, newCustomMulai, newCustomSelesai string) error {
	// Parse waktu custom yang baru dengan format "15:04:05"
	newMulai, err := time.Parse("15:04:05", newCustomMulai)
	if err != nil {
		return fmt.Errorf("format custom_jam_mulai tidak valid: %v", err)
	}
	newSelesai, err := time.Parse("15:04:05", newCustomSelesai)
	if err != nil {
		return fmt.Errorf("format custom_jam_selesai tidak valid: %v", err)
	}

	// Ambil default waktu shift dari tabel Shift berdasarkan id_shift dari Shift_Karyawan
	var shiftJamMulai, shiftJamSelesai string
	query := `
		SELECT s.jam_mulai, s.jam_selesai 
		FROM Shift_Karyawan sk
		JOIN Shift s ON sk.id_shift = s.id_shift
		WHERE sk.id_shift_karyawan = ?
	`
	err = s.DB.QueryRow(query, idShiftKaryawan).Scan(&shiftJamMulai, &shiftJamSelesai)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("record Shift_Karyawan dengan id %d tidak ditemukan", idShiftKaryawan)
		}
		return err
	}

	// Parse default waktu shift
	defaultMulai, err := time.Parse("15:04:05", shiftJamMulai)
	if err != nil {
		return fmt.Errorf("format default jam_mulai tidak valid: %v", err)
	}
	defaultSelesai, err := time.Parse("15:04:05", shiftJamSelesai)
	if err != nil {
		return fmt.Errorf("format default jam_selesai tidak valid: %v", err)
	}

	// Validasi: custom_jam_mulai tidak boleh sebelum default dan custom_jam_selesai tidak boleh melewati default
	if newMulai.Before(defaultMulai) || newSelesai.After(defaultSelesai) {
		return fmt.Errorf("custom shift harus berada dalam rentang waktu %s - %s", shiftJamMulai, shiftJamSelesai)
	}

	// Update record Shift_Karyawan dengan waktu custom yang baru
	updateQuery := `
		UPDATE Shift_Karyawan 
		SET custom_jam_mulai = ?, custom_jam_selesai = ?
		WHERE id_shift_karyawan = ?
	`
	res, err := s.DB.Exec(updateQuery, newCustomMulai, newCustomSelesai, idShiftKaryawan)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("tidak ada record yang diupdate")
	}
	return nil
}

func (s *ShiftService) SoftDeleteShiftKaryawan(idShiftKaryawan int, idManagement int) error {
	// Update deleted_by dari NULL menjadi idManagement untuk record yang belum dihapus
	updateQuery := `
		UPDATE Management_Shift_Karyawan 
		SET deleted_by = ? 
		WHERE id_shift_karyawan = ? AND deleted_by IS NULL
	`
	res, err := s.DB.Exec(updateQuery, idManagement, idShiftKaryawan)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tidak ada record yang di soft delete atau record sudah di soft delete")
	}
	return nil
}

func (s *ShiftService) GetShiftPoliList(idPoliFilter string) ([]map[string]interface{}, error) {
	// Kondisi join hanya berdasarkan id_shift dan tanggal, tanpa filter CURTIME()
	joinCondition := "s.id_shift = sk.id_shift AND sk.tanggal = CURDATE()"
	var args []interface{}

	// Jika ada filter berdasarkan id_poli, tambahkan kondisi
	if idPoliFilter != "" {
		joinCondition += " AND sk.id_poli = ?"
		args = append(args, idPoliFilter)
	}

	query := fmt.Sprintf(`
		SELECT 
			s.id_shift, 
			s.jam_mulai, 
			s.jam_selesai,
			CASE 
				WHEN s.id_shift = 1 THEN 'Shift Pagi'
				WHEN s.id_shift = 2 THEN 'Shift Sore'
				ELSE 'Shift Lainnya'
			END AS nama_shift,
			COUNT(DISTINCT sk.id_karyawan) AS jumlah_tenkes
		FROM Shift s
		LEFT JOIN Shift_Karyawan sk 
			ON %s
		GROUP BY s.id_shift, s.jam_mulai, s.jam_selesai
		ORDER BY s.id_shift
	`, joinCondition)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var idShift int
		var jamMulai, jamSelesai, namaShift string
		var jumlahTenkes int
		if err := rows.Scan(&idShift, &jamMulai, &jamSelesai, &namaShift, &jumlahTenkes); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}
		record := map[string]interface{}{
			"id_shift":      idShift,
			"nama_shift":    namaShift,
			"jam_mulai":     jamMulai,
			"jam_selesai":   jamSelesai,
			"jumlah_tenkes": jumlahTenkes,
		}
		results = append(results, record)
	}
	return results, nil
}

// GetListKaryawanFiltered mengambil daftar karyawan berdasarkan:
// - id_poli (wajib)
// - id_shift (wajib)
// - id_role (opsional; jika tidak ada, tampilkan semua)
func (s *ShiftService) GetListKaryawanFiltered(idPoliFilter, idShiftFilter, idRoleFilter string) ([]map[string]interface{}, error) {
	// Pastikan id_poli dan id_shift wajib ada.
	idPoli, err := strconv.Atoi(idPoliFilter)
	if err != nil {
		return nil, fmt.Errorf("invalid id_poli value: %v", err)
	}
	idShift, err := strconv.Atoi(idShiftFilter)
	if err != nil {
		return nil, fmt.Errorf("invalid id_shift value: %v", err)
	}

	baseQuery := `
		SELECT 
			k.id_karyawan,
			k.nama,
			k.nik,
			k.username,
			k.no_telp,
			k.alamat,
			sk.custom_jam_mulai,
			sk.custom_jam_selesai,
			GROUP_CONCAT(drk.id_role) AS roles
		FROM Karyawan k
		JOIN Shift_Karyawan sk ON k.id_karyawan = sk.id_karyawan 
			AND sk.id_poli = ? 
			AND sk.id_shift = ? 
			AND sk.tanggal = CURDATE()
	`
	args := []interface{}{idPoli, idShift}

	// Jika id_role filter diberikan, tambahkan kondisi di WHERE.
	if idRoleFilter != "" {
		idRole, err := strconv.Atoi(idRoleFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid id_role value: %v", err)
		}
		baseQuery += " AND k.id_karyawan IN (SELECT id_karyawan FROM Detail_Role_Karyawan WHERE id_role = ?)"
		args = append(args, idRole)
	}

	baseQuery += `
		LEFT JOIN Detail_Role_Karyawan drk ON k.id_karyawan = drk.id_karyawan
		GROUP BY k.id_karyawan, k.nama, k.nik, k.username, k.no_telp, k.alamat, sk.custom_jam_mulai, sk.custom_jam_selesai
		ORDER BY k.id_karyawan
	`

	rows, err := s.DB.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var idKaryawan int
		var nama, nik, username, noTelp, alamat string
		var customJamMulai, customJamSelesai string
		var rolesStr sql.NullString

		if err := rows.Scan(&idKaryawan, &nama, &nik, &username, &noTelp, &alamat, &customJamMulai, &customJamSelesai, &rolesStr); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}

		var roles []int
		if rolesStr.Valid && rolesStr.String != "" {
			parts := strings.Split(rolesStr.String, ",")
			for _, part := range parts {
				if r, err := strconv.Atoi(part); err == nil {
					roles = append(roles, r)
				}
			}
		}

		record := map[string]interface{}{
			"id_karyawan":       idKaryawan,
			"nama":              nama,
			"NIK":               nik,
			"username":          username,
			"role":              roles,
			"no_telp":           noTelp,
			"alamat":            alamat,
			"custom_jam_mulai":  customJamMulai,
			"custom_jam_selesai": customJamSelesai,
		}
		results = append(results, record)
	}

	return results, nil
}
