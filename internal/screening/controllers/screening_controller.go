package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/screening/models"
	"github.com/c14220110/poliklinik-backend/internal/screening/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
)

// ScreeningController menangani endpoint terkait screening.
type ScreeningController struct {
	Service *services.ScreeningService
}

// NewScreeningController membuat instance ScreeningController.
func NewScreeningController(service *services.ScreeningService) *ScreeningController {
	return &ScreeningController{Service: service}
}

// InputScreeningRequest mendefinisikan struktur data request untuk input screening.
type InputScreeningRequest struct {
	ID_Pasien      int     `json:"id_pasien"`
	Tensi          string  `json:"tensi"`
	Berat_Badan    int     `json:"berat_badan"`
	Suhu_Tubuh     float64 `json:"suhu_tubuh"`
	Tinggi_Badan   float64 `json:"tinggi_badan"`
	Gula_Darah     float64 `json:"gula_darah"`
	Detak_Nadi     int     `json:"detak_nadi"`
	Laju_Respirasi int     `json:"laju_respirasi"`
	Keterangan     string  `json:"keterangan"`
}

// InputScreening menerima data screening, menyimpannya ke tabel Screening,
// dan mengupdate record Riwayat_Kunjungan yang belum memiliki ID_Screening.
// Nilai operator (ID_Karyawan) diambil dari token JWT terpadu yang disimpan di context.
func (sc *ScreeningController) InputScreening(w http.ResponseWriter, r *http.Request) {
	// Ambil klaim terpadu dari context
	claims, ok := r.Context().Value(middlewares.ContextKeyUserID).(*utils.Claims)
	if !ok || claims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing operator ID in token",
			"data":    nil,
		})
		return
	}

	// Konversi claims.IDKaryawan (tipe string) ke int
	operatorID, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || operatorID <= 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid operator ID in token",
			"data":    nil,
		})
		return
	}

	var req InputScreeningRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}
	if req.ID_Pasien <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "ID_Pasien must be provided and greater than 0",
			"data":    nil,
		})
		return
	}

	// Buat objek Screening
	screening := models.Screening{
		ID_Pasien:      req.ID_Pasien,
		ID_Karyawan:    operatorID,
		Tensi:          req.Tensi,
		Berat_Badan:    req.Berat_Badan,
		Suhu_Tubuh:     req.Suhu_Tubuh,
		Tinggi_Badan:   req.Tinggi_Badan,
		Gula_Darah:     req.Gula_Darah,
		Detak_Nadi:     req.Detak_Nadi,
		Laju_Respirasi: req.Laju_Respirasi,
		Keterangan:     req.Keterangan,
		Created_At:     time.Now(),
	}

	// Panggil service untuk memasukkan data screening dan update Riwayat_Kunjungan
	screeningID, err := sc.Service.InputScreening(screening)
	if err != nil {
		// Jika error karena tidak ditemukan record Riwayat_Kunjungan pending, artinya screening sudah tercatat
		if err.Error() == "failed to get Riwayat_Kunjungan: sql: no rows in result set" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusConflict,
				"message": "Screening has already been recorded for this visit",
				"data":    nil,
			})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to input screening: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Screening recorded successfully",
		"data": map[string]interface{}{
			"ID_Screening": screeningID,
		},
	})
}

// GetScreeningByPasienHandler mengembalikan seluruh record screening untuk pasien berdasarkan query parameter id_pasien.
func (sc *ScreeningController) GetScreeningByPasienHandler(w http.ResponseWriter, r *http.Request) {
	idPasienParam := r.URL.Query().Get("id_pasien")
	if idPasienParam == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_pasien parameter is required",
			"data":    nil,
		})
		return
	}

	idPasien, err := strconv.Atoi(idPasienParam)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_pasien must be a number",
			"data":    nil,
		})
		return
	}

	screenings, err := sc.Service.GetScreeningByPasien(idPasien)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve screening records: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Screening records retrieved successfully",
		"data":    screenings,
	})
}