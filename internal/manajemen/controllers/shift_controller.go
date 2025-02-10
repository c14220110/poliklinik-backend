package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
)

type ShiftController struct {
	Service *services.ShiftService
	DB      *sql.DB // untuk mengambil data poliklinik
}

func NewShiftController(service *services.ShiftService, db *sql.DB) *ShiftController {
	return &ShiftController{
		Service: service,
		DB:      db,
	}
}

// GetPoliklinikListHandler returns the list of poliklinik.
func (sc *ShiftController) GetPoliklinikListHandler(w http.ResponseWriter, r *http.Request) {
	data, err := sc.Service.GetPoliklinikList()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve poliklinik list: " + err.Error(),
			"data":    nil,
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Poliklinik list retrieved successfully",
		"data":    data,
	})
}

func (sc *ShiftController) GetKaryawanByShiftAndPoliHandler(w http.ResponseWriter, r *http.Request) {
	shiftIDParam := r.URL.Query().Get("shift_id")
	poliIDParam := r.URL.Query().Get("poli_id")
	if shiftIDParam == "" || poliIDParam == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Both shift_id and poli_id parameters are required",
			"data":    nil,
		})
		return
	}

	shiftID, err := strconv.Atoi(shiftIDParam)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "shift_id must be a number",
			"data":    nil,
		})
		return
	}

	poliID, err := strconv.Atoi(poliIDParam)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "poli_id must be a number",
			"data":    nil,
		})
		return
	}

	data, err := sc.Service.GetKaryawanByShiftAndPoli(shiftID, poliID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve karyawan by shift: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Karyawan retrieved successfully",
		"data":    data,
	})
}

// GetShiftSummaryHandler mengembalikan ringkasan shift berdasarkan poli_id yang diberikan.
func (sc *ShiftController) GetShiftSummaryHandler(w http.ResponseWriter, r *http.Request) {
	poliIDParam := r.URL.Query().Get("poli_id")
	if poliIDParam == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "poli_id parameter is required",
			"data":    nil,
		})
		return
	}
	poliID, err := strconv.Atoi(poliIDParam)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "poli_id must be a number",
			"data":    nil,
		})
		return
	}

	data, err := sc.Service.GetShiftSummaryByPoli(poliID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve shift summary: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Shift summary retrieved successfully",
		"data":    data,
	})
}