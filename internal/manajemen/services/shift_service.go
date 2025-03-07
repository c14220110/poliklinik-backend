package services

import (
	"database/sql"
	"fmt"
	"time"
)

type ShiftService struct {
	DB *sql.DB
}

func NewShiftService(db *sql.DB) *ShiftService {
	return &ShiftService{DB: db}
}

// AssignShift melakukan insert ke tabel Shift_Karyawan dan Management_Shift_Karyawan.
// Parameter id_management berasal dari JWT (dari management yang membuat record).
func (s *ShiftService) AssignShift(idPoli, idKaryawan, idRole, idShift, idManagement int, tanggalStr string) (int64, error) {
    // Parse tanggal dengan format "2006-01-02"
    tanggal, err := time.Parse("2006-01-02", tanggalStr)
    if err != nil {
        return 0, fmt.Errorf("format tanggal tidak valid: %v", err)
    }

    // Ambil data Shift untuk mendapatkan jam_mulai dan jam_selesai default
    var jamMulai, jamSelesai string
    queryShift := "SELECT jam_mulai, jam_selesai FROM Shift WHERE id_shift = ?"
    err = s.DB.QueryRow(queryShift, idShift).Scan(&jamMulai, &jamSelesai)
    if err != nil {
        if err == sql.ErrNoRows {
            return 0, fmt.Errorf("id_shift %d tidak ditemukan", idShift)
        }
        return 0, err
    }

    // Mulai transaction
    tx, err := s.DB.Begin()
    if err != nil {
        return 0, err
    }

    // Insert ke tabel Shift_Karyawan
    insertQuery := `
        INSERT INTO Shift_Karyawan (id_poli, id_shift, id_karyawan, custom_jam_mulai, custom_jam_selesai, tanggal)
        VALUES (?, ?, ?, ?, ?, ?)
    `
    res, err := tx.Exec(insertQuery, idPoli, idShift, idKaryawan, jamMulai, jamSelesai, tanggal)
    if err != nil {
        tx.Rollback()
        return 0, err
    }
    idShiftKaryawan, err := res.LastInsertId()
    if err != nil {
        tx.Rollback()
        return 0, err
    }

    // Insert ke tabel Management_Shift_Karyawan
    insertManagementShiftQuery := `
        INSERT INTO Management_Shift_Karyawan (id_management, id_shift_karyawan, created_by, updated_by, deleted_by)
        VALUES (?, ?, ?, ?, ?)
    `
    _, err = tx.Exec(insertManagementShiftQuery, idManagement, idShiftKaryawan, idManagement, idManagement, 0)
    if err != nil {
        tx.Rollback()
        return 0, err
    }

    // Commit transaction
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

func (ps *ShiftService) GetShiftPoliList() ([]map[string]interface{}, error) {
	// Query ini mengambil data shift dan menghitung jumlah tenaga kesehatan yang aktif di masing-masing shift hari ini.
	query := `
		SELECT 
			s.id_shift, 
			s.jam_mulai, 
			s.jam_selesai,
			CASE 
				WHEN s.id_shift = 1 THEN 'Shift Pagi'
				WHEN s.id_shift = 2 THEN 'Shift Sore'
				ELSE 'Shift Lainnya'
			END AS nama_shift,
			COUNT(sk.id_karyawan) AS jumlah_tenkes
		FROM Shift s
		LEFT JOIN Shift_Karyawan sk 
			ON s.id_shift = sk.id_shift 
			AND sk.tanggal = CURDATE() 
			AND CURTIME() BETWEEN sk.custom_jam_mulai AND sk.custom_jam_selesai
		GROUP BY s.id_shift, s.jam_mulai, s.jam_selesai
		ORDER BY s.id_shift
	`
	rows, err := ps.DB.Query(query)
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