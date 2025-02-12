package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type PoliklinikController struct {
	DB *sql.DB
}

func NewPoliklinikController(db *sql.DB) *PoliklinikController {
	return &PoliklinikController{DB: db}
}

func (pc *PoliklinikController) GetPoliklinikList(w http.ResponseWriter, r *http.Request) {
	query := "SELECT ID_Poli, Nama_Poli FROM Poliklinik ORDER BY Nama_Poli ASC"
	rows, err := pc.DB.Query(query)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve poliklinik list: " + err.Error(),
			"data":    nil,
		})
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var idPoli int
		var namaPoli string
		if err := rows.Scan(&idPoli, &namaPoli); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to scan row: " + err.Error(),
				"data":    nil,
			})
			return
		}
		results = append(results, map[string]interface{}{
			"ID_Poli":   idPoli,
			"Nama_Poli": namaPoli,
		})
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Poliklinik list retrieved successfully",
		"data":    results,
	})
}
