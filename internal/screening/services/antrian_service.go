package services

import (
	"database/sql"
	"errors"
	"fmt"
)

type AntrianService struct {
	DB *sql.DB
}

func NewAntrianService(db *sql.DB) *AntrianService {
	return &AntrianService{DB: db}
}

// MasukkanPasien mencari baris antrian dengan id_status = 0 (misal, "Menunggu")
// untuk poli dengan idPoli tertentu dan mengubah id_status menjadi 2.
func (s *AntrianService) MasukkanPasien(idPoli int) error {
	// Cari baris antrian teratas dengan id_status = 0 untuk id_poli yang diberikan.
	query := `
		SELECT id_antrian 
		FROM Antrian 
		WHERE id_poli = ? AND id_status = 1 
		ORDER BY nomor_antrian ASC 
		LIMIT 1
	`
	var idAntrian int
	err := s.DB.QueryRow(query, idPoli).Scan(&idAntrian)
	if err != nil {
		if err == sql.ErrNoRows {
			// Tidak ditemukan baris dengan status 1 atau id_poli tidak ada.
			return fmt.Errorf("tidak ada pasien dengan status 1 untuk poli dengan id %d", idPoli)
		}
		return err
	}

	// Update baris yang ditemukan, ubah id_status menjadi 3.
	updateQuery := `
		UPDATE Antrian 
		SET id_status = 3 
		WHERE id_antrian = ?
	`
	res, err := s.DB.Exec(updateQuery, idAntrian)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("gagal mengupdate antrian, baris tidak ditemukan")
	}
	return nil
}


// GetAntrianTerlama mengambil ID_Antrian dan Nomor_Antrian dari pasien dengan antrian paling lama (status = 0)
func (s *AntrianService) GetAntrianTerlama(idPoli int) (map[string]interface{}, error) {
	query := `
		SELECT id_antrian, nomor_antrian 
    FROM Antrian 
    WHERE id_poli = ? AND id_status = 1 
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
