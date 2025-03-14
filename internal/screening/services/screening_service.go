package services

import (
	"database/sql"
	"fmt"

	"github.com/c14220110/poliklinik-backend/internal/screening/models"
)

type ScreeningService struct {
	DB *sql.DB
}

func NewScreeningService(db *sql.DB) *ScreeningService {
	return &ScreeningService{DB: db}
}

func (s *ScreeningService) InputScreening(screening models.Screening, idAntrian int) (int64, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}

	// 1. Insert data screening ke tabel Screening (sesuai schema baru)
	queryScreening := `
		INSERT INTO Screening (
			ID_Pasien, 
			ID_Karyawan, 
			systolic, 
			diastolic, 
			berat_badan, 
			suhu_tubuh, 
			tinggi_badan, 
			detak_nadi, 
			laju_respirasi, 
			keterangan
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
	)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	screeningID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// 2. Update Riwayat_Kunjungan record (gunakan query parameter id_antrian)
	updateRK := `
		UPDATE Riwayat_Kunjungan
		SET ID_Screening = ?
		WHERE id_antrian = ? AND ID_Screening IS NULL
	`
	res, err = tx.Exec(updateRK, screeningID, idAntrian)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	if affected == 0 {
		tx.Rollback()
		return 0, fmt.Errorf("failed to update Riwayat_Kunjungan with screening ID")
	}

	// 3. Update status di tabel Antrian (set id_status menjadi 4)
	updateAntrian := `
		UPDATE Antrian 
		SET id_status = 4 
		WHERE id_antrian = ?
	`
	res, err = tx.Exec(updateAntrian, idAntrian)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to update Antrian status: %v", err)
	}
	affected, err = res.RowsAffected()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	if affected == 0 {
		tx.Rollback()
		return 0, fmt.Errorf("failed to update Antrian status")
	}

	if err = tx.Commit(); err != nil {
		return 0, err
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
