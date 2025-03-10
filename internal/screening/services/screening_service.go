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

func (s *ScreeningService) InputScreening(screening models.Screening) (int64, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}

	// 1. Insert data screening ke tabel Screening
	queryScreening := `
		INSERT INTO Screening (ID_Pasien, ID_Karyawan, Tensi, Berat_Badan, Suhu_Tubuh, Tinggi_Badan, Gula_Darah, Detak_Nadi, Laju_Respirasi, Keterangan)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := tx.Exec(queryScreening,
		screening.ID_Pasien,
		screening.ID_Karyawan,
		screening.Tensi,
		screening.Berat_Badan,
		screening.Suhu_Tubuh,
		screening.Tinggi_Badan,
		screening.Gula_Darah,
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

	// 2. Cari record Riwayat_Kunjungan untuk pasien yang belum diisi screening.
	// Ambil ID_RM dari Rekam_Medis untuk pasien tersebut (record terbaru)
	var idRM int64
	err = tx.QueryRow(`
		SELECT ID_RM FROM Rekam_Medis 
		WHERE ID_Pasien = ? 
		ORDER BY Created_At DESC LIMIT 1
	`, screening.ID_Pasien).Scan(&idRM)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to get ID_RM: %v", err)
	}

	// Cari record Riwayat_Kunjungan dengan ID_RM tersebut dan di mana ID_Screening masih NULL,
	// ambil record terbaru.
	var idKunjungan int64
	err = tx.QueryRow(`
		SELECT ID_Kunjungan FROM Riwayat_Kunjungan 
		WHERE ID_RM = ? AND ID_Screening IS NULL 
		ORDER BY Created_At DESC LIMIT 1
	`, idRM).Scan(&idKunjungan)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to get Riwayat_Kunjungan: %v", err)
	}

	// 3. Update record Riwayat_Kunjungan dengan memasukkan ID_Screening yang baru dibuat.
	_, err = tx.Exec(`
		UPDATE Riwayat_Kunjungan 
		SET ID_Screening = ? 
		WHERE ID_Kunjungan = ?
	`, screeningID, idKunjungan)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// 4. Update status di tabel Antrian: ambil id_antrian dari Riwayat_Kunjungan dan set id_status menjadi 4
	var idAntrian int64
	err = tx.QueryRow(`
		SELECT ID_Antrian FROM Riwayat_Kunjungan 
		WHERE ID_Kunjungan = ?
	`, idKunjungan).Scan(&idAntrian)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to get id_antrian from Riwayat_Kunjungan: %v", err)
	}

	_, err = tx.Exec(`
		UPDATE Antrian 
		SET id_status = 4 
		WHERE id_antrian = ?
	`, idAntrian)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to update Antrian status: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return screeningID, nil
}

// GetScreeningByPasien mengembalikan daftar record screening untuk pasien dengan ID_Pasien tertentu.
func (s *ScreeningService) GetScreeningByPasien(idPasien int) ([]models.Screening, error) {
	query := `
		SELECT ID_Screening, ID_Pasien, ID_Karyawan, Tensi, Berat_Badan, Suhu_Tubuh, 
		       Tinggi_Badan, Gula_Darah, Detak_Nadi, Laju_Respirasi, Keterangan, Created_At
		FROM Screening
		WHERE ID_Pasien = ?
		ORDER BY Created_At DESC
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
			&rec.Tensi,
			&rec.Berat_Badan,
			&rec.Suhu_Tubuh,
			&rec.Tinggi_Badan,
			&rec.Gula_Darah,
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