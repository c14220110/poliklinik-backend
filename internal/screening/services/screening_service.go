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

// InputScreening menyimpan data screening dan mengaitkan record screening tersebut ke record Riwayat_Kunjungan
// untuk pasien yang belum diisi screening (ID_Screening masih NULL) pada kunjungan terbaru.
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
	// Pertama, ambil ID_RM dari Rekam_Medis untuk pasien tersebut (record terbaru)
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

	// Kemudian, cari record Riwayat_Kunjungan dengan ID_RM tersebut dan di mana ID_Screening masih NULL,
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

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return screeningID, nil
}
