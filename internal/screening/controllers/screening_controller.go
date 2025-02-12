package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/screening/models"
	"github.com/c14220110/poliklinik-backend/internal/screening/services"
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
// Nilai operator (ID_Karyawan) diambil dari token JWT yang disimpan ke context.
func (sc *ScreeningController) InputScreening(w http.ResponseWriter, r *http.Request) {
	// Ekstrak operator ID dari context menggunakan key yang didefinisikan di middleware.
	operatorID, ok := r.Context().Value(middlewares.ContextKeyUserID).(int)
	if !ok || operatorID <= 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing operator ID in token",
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
			"message": "ID_Pasien is required and must be greater than 0",
			"data":    nil,
		})
		return
	}

	// Buat objek Screening berdasarkan request
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
		Created_At:     time.Now(), // Dapat juga diatur default oleh DB
	}

	// Panggil fungsi service untuk menyimpan data screening dan update Riwayat_Kunjungan.
	screeningID, err := sc.Service.InputScreening(screening)
	if err != nil {
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
