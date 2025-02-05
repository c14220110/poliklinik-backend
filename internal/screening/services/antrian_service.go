package services

import (
	"database/sql"
	"errors"
	//"time"
)

// AntrianService menangani logika bisnis untuk antrian screening.
type AntrianService struct {
	DB *sql.DB
}

func NewAntrianService(db *sql.DB) *AntrianService {
	return &AntrianService{DB: db}
}

// GetAntrianTerlama mengambil ID_Antrian dan Nomor_Antrian dari pasien dengan antrian paling lama (status = 0)
func (s *AntrianService) GetAntrianTerlama(idPoli int) (map[string]interface{}, error) {
	query := `
		SELECT ID_Antrian, Nomor_Antrian
		FROM Antrian
		WHERE ID_Poli = ? AND Status = 0
		ORDER BY Created_At ASC
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
