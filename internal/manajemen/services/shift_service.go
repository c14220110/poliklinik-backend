package services

import (
	"database/sql"
)

type ShiftService struct {
	DB *sql.DB
}

func NewShiftService(db *sql.DB) *ShiftService {
	return &ShiftService{DB: db}
}

// GetPoliklinikList returns list of poliklinik (ID_Poli and Nama_Poli).
func (s *ShiftService) GetPoliklinikList() ([]map[string]interface{}, error) {
	query := "SELECT ID_Poli, Nama_Poli FROM Poliklinik ORDER BY Nama_Poli ASC"
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var idPoli int
		var namaPoli string
		if err := rows.Scan(&idPoli, &namaPoli); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"ID_Poli":   idPoli,
			"Nama_Poli": namaPoli,
		})
	}
	return results, nil
}

// GetKaryawanByShiftAndPoli mengembalikan daftar karyawan berdasarkan shift dan poliklinik.
func (s *ShiftService) GetKaryawanByShiftAndPoli(shiftID int, poliID int) ([]map[string]interface{}, error) {
	query := `
		SELECT k.ID_Karyawan, k.Nama, k.Username, k.No_Telp, sk.Tanggal
		FROM Shift_Karyawan sk
		JOIN Karyawan k ON sk.ID_Karyawan = k.ID_Karyawan
		WHERE sk.ID_Shift = ? AND sk.ID_Poli = ?
		ORDER BY sk.Tanggal DESC
	`
	rows, err := s.DB.Query(query, shiftID, poliID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var idKaryawan int
		var nama, username, noTelp string
		var tanggal string // atau time.Time, tergantung kebutuhan; di sini gunakan string
		if err := rows.Scan(&idKaryawan, &nama, &username, &noTelp, &tanggal); err != nil {
			return nil, err
		}
		record := map[string]interface{}{
			"ID_Karyawan": idKaryawan,
			"Nama":        nama,
			"Username":    username,
			"No_Telp":     noTelp,
			"Tanggal":     tanggal,
		}
		results = append(results, record)
	}
	return results, nil
}

// GetShiftSummaryByPoli mengembalikan ringkasan shift untuk suatu poliklinik.
// Hasilnya mencakup Nama_Shift, Jam_Mulai, Jam_Selesai, dan Jumlah_Tenaga_Kerja.
func (s *ShiftService) GetShiftSummaryByPoli(poliID int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			s.ID_Shift,
			CASE 
				WHEN s.Jam_Mulai BETWEEN '08:00:00' AND '14:59:59' THEN 'Shift Pagi'
				WHEN s.Jam_Mulai BETWEEN '15:00:00' AND '22:59:59' THEN 'Shift Sore'
				ELSE 'Shift Malam'
			END AS Nama_Shift,
			s.Jam_Mulai,
			s.Jam_Selesai,
			COUNT(sk.ID_Karyawan) AS Jumlah_Tenaga_Kerja
		FROM Shift s
		JOIN Shift_Karyawan sk ON s.ID_Shift = sk.ID_Shift
		WHERE sk.ID_Poli = ?
		GROUP BY s.ID_Shift, s.Jam_Mulai, s.Jam_Selesai
		ORDER BY s.Jam_Mulai ASC
	`
	rows, err := s.DB.Query(query, poliID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var idShift int
		var namaShift string
		var jamMulai, jamSelesai string
		var jumlah int
		if err := rows.Scan(&idShift, &namaShift, &jamMulai, &jamSelesai, &jumlah); err != nil {
			return nil, err
		}
		record := map[string]interface{}{
			"Nama_Shift":             namaShift,
			"Jam_Mulai":              jamMulai,
			"Jam_Selesai":            jamSelesai,
			"Jumlah_Tenaga_Kerja":    jumlah,
		}
		results = append(results, record)
	}
	return results, nil
}