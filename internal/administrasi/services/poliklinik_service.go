package services

import (
	"database/sql"
)

type PoliklinikService struct {
	DB *sql.DB
}

func NewPoliklinikService(db *sql.DB) *PoliklinikService {
	return &PoliklinikService{DB: db}
}

// GetPoliklinikList mengembalikan daftar poliklinik dengan field ID_Poli dan Nama_Poli.
func (ps *PoliklinikService) GetPoliklinikList() ([]map[string]interface{}, error) {
	query := "SELECT ID_Poli, Nama_Poli FROM Poliklinik ORDER BY Nama_Poli ASC"
	rows, err := ps.DB.Query(query)
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
