package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/screening/services"
)

type AntrianController struct {
	Service *services.AntrianService
}

func NewAntrianController(service *services.AntrianService) *AntrianController {
	return &AntrianController{Service: service}
}

// GetAntrianTerlamaHandler menangani request untuk mendapatkan antrian pasien paling lama dengan status = 0
func (ac *AntrianController) GetAntrianTerlamaHandler(w http.ResponseWriter, r *http.Request) {
	idPoliParam := r.URL.Query().Get("id_poli")
	if idPoliParam == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli harus diberikan",
			"data":    nil,
		})
		return
	}

	idPoli, err := strconv.Atoi(idPoliParam)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli harus berupa angka",
			"data":    nil,
		})
		return
	}

	data, err := ac.Service.GetAntrianTerlama(idPoli)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusNotFound,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Antrian ditemukan",
		"data":    data,
	})
}
