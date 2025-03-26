package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/screening/models"
)

type ScreeningService struct {
	DB *sql.DB
}

func NewScreeningService(db *sql.DB) *ScreeningService {
	return &ScreeningService{DB: db}
}

func (s *ScreeningService) InputScreening(input models.ScreeningInput, idAntrian int, operatorID int) (int64, error) {
	// Mulai transaksi
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("gagal memulai transaksi: %v", err)
	}

	// Ambil id_pasien dari Antrian berdasarkan id_antrian di Riwayat_Kunjungan
	var idPasien int
	queryGetPasien := `
		SELECT A.id_pasien 
		FROM Antrian A
		JOIN Riwayat_Kunjungan RK ON A.id_antrian = RK.id_antrian
		WHERE RK.id_antrian = ?
	`
	err = tx.QueryRow(queryGetPasien, idAntrian).Scan(&idPasien)
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("tidak ditemukan Antrian untuk id_antrian %d", idAntrian)
		}
		return 0, fmt.Errorf("gagal mengambil id_pasien: %v", err)
	}

	// Buat objek Screening
	screening := models.Screening{
		ID_Pasien:      idPasien,
		ID_Karyawan:    operatorID,
		Systolic:       input.Systolic,
		Diastolic:      input.Diastolic,
		Berat_Badan:    input.Berat_Badan,
		Suhu_Tubuh:     input.Suhu_Tubuh,
		Tinggi_Badan:   input.Tinggi_Badan,
		Detak_Nadi:     input.Detak_Nadi,
		Laju_Respirasi: input.Laju_Respirasi,
		Keterangan:     input.Keterangan,
		Created_At:     time.Now(),
	}

	// Insert data screening ke tabel Screening
	queryScreening := `
		INSERT INTO Screening (
			id_pasien, 
			id_karyawan, 
			systolic, 
			diastolic, 
			berat_badan, 
			suhu_tubuh, 
			tinggi_badan, 
			detak_nadi, 
			laju_respirasi, 
			keterangan,
			created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := tx.Exec(queryScreening,
		screening.ID_Pasien,
		screening.ID_Karyawan,
		screening.Systolic,
		screening.Diastolic,
		screening.Berat_Badan,
		screening.Suhu_Tubuh,
		screening.Tinggi_Badan,
		screening.Detak_Nadi,
		screening.Laju_Respirasi,
		screening.Keterangan,
		screening.Created_At,
	)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("gagal insert data screening: %v", err)
	}
	screeningID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("gagal mendapatkan ID screening: %v", err)
	}

	// Update Riwayat_Kunjungan dengan id_screening
	updateRK := `
		UPDATE Riwayat_Kunjungan
		SET id_screening = ?
		WHERE id_antrian = ? AND id_screening IS NULL
	`
	res, err = tx.Exec(updateRK, screeningID, idAntrian)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("gagal update Riwayat_Kunjungan: %v", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("gagal memeriksa affected rows: %v", err)
	}
	if affected == 0 {
		tx.Rollback()
		return 0, fmt.Errorf("gagal update Riwayat_Kunjungan dengan ID screening")
	}

	// Update status Antrian menjadi 4 (misalnya Pra-Konsultasi)
	updateAntrian := `
		UPDATE Antrian 
		SET id_status = 4 
		WHERE id_antrian = ?
	`
	res, err = tx.Exec(updateAntrian, idAntrian)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("gagal update status Antrian: %v", err)
	}
	affected, err = res.RowsAffected()
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("gagal memeriksa affected rows Antrian: %v", err)
	}
	if affected == 0 {
		tx.Rollback()
		return 0, fmt.Errorf("gagal update status Antrian")
	}

	// Commit transaksi
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("gagal commit transaksi: %v", err)
	}

	return screeningID, nil
}

// GetScreeningByPasien mengembalikan daftar record screening untuk pasien dengan ID_Pasien tertentu.
func (s *ScreeningService) GetScreeningByPasien(idPasien int) ([]models.Screening, error) {
	query := `
		SELECT id_screening, id_pasien, id_karyawan, systolic, diastolic, berat_badan, suhu_tubuh, 
		       tinggi_badan, detak_nadi, laju_respirasi, keterangan, created_at
		FROM Screening
		WHERE id_pasien = ?
		ORDER BY created_at DESC
	`
	rows, err := s.DB.Query(query, idPasien)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var screenings []models.Screening
	for rows.Next() {
		var rec models.Screening
		err := rows.Scan(
			&rec.ID_Screening,
			&rec.ID_Pasien,
			&rec.ID_Karyawan,
			&rec.Systolic,
			&rec.Diastolic,
			&rec.Berat_Badan,
			&rec.Suhu_Tubuh,
			&rec.Tinggi_Badan,
			&rec.Detak_Nadi,
			&rec.Laju_Respirasi,
			&rec.Keterangan,
			&rec.Created_At,
		)
		if err != nil {
			return nil, err
		}
		screenings = append(screenings, rec)
	}
	return screenings, nil
}
