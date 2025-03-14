package services

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type AntrianService struct {
	DB *sql.DB
}

func NewAntrianService(db *sql.DB) *AntrianService {
	return &AntrianService{DB: db}
}

func (s *AntrianService) MasukkanPasien(idPoli int) (map[string]interface{}, error) {
	// 1. Cari baris antrian teratas dengan id_status = 1 untuk id_poli yang diberikan dan untuk hari ini.
	query := `
		SELECT id_antrian 
		FROM Antrian 
		WHERE id_poli = ? AND id_status = 1 AND DATE(created_at) = CURDATE()
		ORDER BY nomor_antrian ASC 
		LIMIT 1
	`
	var idAntrian int
	err := s.DB.QueryRow(query, idPoli).Scan(&idAntrian)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tidak ada pasien dengan status 1 untuk poli dengan id %d pada hari ini", idPoli)
		}
		return nil, err
	}

	// 2. Update baris yang ditemukan, ubah id_status menjadi 3.
	updateQuery := `
		UPDATE Antrian 
		SET id_status = 3 
		WHERE id_antrian = ?
	`
	res, err := s.DB.Exec(updateQuery, idAntrian)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, fmt.Errorf("gagal mengupdate antrian, baris tidak ditemukan")
	}

	// 3. Ambil data tambahan: id_pasien, nama pasien, jenis_kelamin, id_rm, tanggal_lahir, dan nomor_antrian
	queryDetails := `
		SELECT p.id_pasien, p.nama, p.jenis_kelamin, rm.id_rm, p.tanggal_lahir, a.nomor_antrian
		FROM Antrian a
		JOIN Pasien p ON a.id_pasien = p.id_pasien
		JOIN Rekam_Medis rm ON p.id_pasien = rm.id_pasien
		WHERE a.id_antrian = ?
		ORDER BY rm.created_at DESC
		LIMIT 1
	`
	var idPasien int
	var nama, jenisKelamin, tanggalLahirStr string
	var idRM int
	var nomorAntrian int

	err = s.DB.QueryRow(queryDetails, idAntrian).Scan(&idPasien, &nama, &jenisKelamin, &idRM, &tanggalLahirStr, &nomorAntrian)
	if err != nil {
		return nil, fmt.Errorf("failed to get detail data: %v", err)
	}

	// Parse tanggal_lahir; gunakan layout RFC3339 karena format di database misalnya "1995-08-15T00:00:00+07:00"
	tanggalLahir, err := time.Parse(time.RFC3339, tanggalLahirStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tanggal_lahir: %v", err)
	}

	// Hitung umur
	now := time.Now()
	umur := now.Year() - tanggalLahir.Year()
	if now.YearDay() < tanggalLahir.YearDay() {
		umur--
	}

	result := map[string]interface{}{
		"id_antrian":     idAntrian,
		"id_pasien":      idPasien,
		"nama_pasien":    nama,
		"jenis_kelamin":  jenisKelamin,
		"id_rm":          idRM,
		"nomor_antrian":  nomorAntrian,
		"umur":           umur,
	}

	return result, nil
}



// GetAntrianTerlama mengambil ID_Antrian dan Nomor_Antrian dari pasien dengan antrian paling lama (status = 1) pada hari ini
func (s *AntrianService) GetAntrianTerlama(idPoli int) (map[string]interface{}, error) {
	query := `
		SELECT id_antrian, nomor_antrian 
		FROM Antrian 
		WHERE id_poli = ? AND id_status = 1 AND DATE(created_at) = CURDATE()
		ORDER BY nomor_antrian ASC 
		LIMIT 1
	`
	var idAntrian int
	var nomorAntrian int

	err := s.DB.QueryRow(query, idPoli).Scan(&idAntrian, &nomorAntrian)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("tidak ada antrian yang menunggu")
		}
		return nil, err
	}

	result := map[string]interface{}{
		"id_antrian":    idAntrian,
		"nomor_antrian": nomorAntrian,
	}

	return result, nil
}
