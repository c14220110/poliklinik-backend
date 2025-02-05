package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/dokter/models"
	"github.com/c14220110/poliklinik-backend/internal/dokter/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
)

type DokterController struct {
	Service *services.DokterService
}

func NewDokterController(service *services.DokterService) *DokterController {
	return &DokterController{Service: service}
}

// CreateDokterRequest adalah struktur request untuk pembuatan dokter.
type CreateDokterRequest struct {
	Nama         string `json:"nama"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Spesialisasi string `json:"spesialisasi"`
}

// CreateDokter menerima request untuk membuat dokter baru.
func (dc *DokterController) CreateDokter(w http.ResponseWriter, r *http.Request) {
	var req CreateDokterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload",
			"data":    nil,
		})
		return
	}
	if req.Nama == "" || req.Username == "" || req.Password == "" || req.Spesialisasi == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Nama, Username, Password, and Spesialisasi are required",
			"data":    nil,
		})
		return
	}

	dokter := models.Dokter{
		Nama:         req.Nama,
		Username:     req.Username,
		Password:     req.Password,
		Spesialisasi: req.Spesialisasi,
	}
	id, err := dc.Service.CreateDokter(dokter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to create dokter: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Dokter created successfully",
		"data": map[string]interface{}{
			"id": id,
		},
	})
}

// LoginDokterRequest adalah struktur request untuk login dokter.
type LoginDokterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IDPoli   int    `json:"id_poli"` // Dokter memilih poliklinik yang ingin diakses
}

// LoginDokter menangani request login dokter dengan validasi shift aktif.
func (dc *DokterController) LoginDokter(w http.ResponseWriter, r *http.Request) {
	var req LoginDokterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}
	if req.Username == "" || req.Password == "" || req.IDPoli == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Username, Password, and IDPoli are required",
			"data":    nil,
		})
		return
	}

	// Autentikasi dokter dengan cek shift aktif
	dokter, shift, err := dc.Service.AuthenticateDokter(req.Username, req.Password, req.IDPoli)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateToken(dokter.ID_Dokter, dokter.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to generate token: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Login successful",
		"data": map[string]interface{}{
			"id":           dokter.ID_Dokter,
			"nama":         dokter.Nama,
			"username":     dokter.Username,
			"spesialisasi": dokter.Spesialisasi,
			"id_poli":      shift.ID_Poli, // ID poli dari shift aktif
			"token":        token,
			"shift": map[string]interface{}{
				"id_shift":    shift.ID_Shift,
				"jam_mulai":   shift.Jam_Mulai,
				"jam_selesai": shift.Jam_Selesai,
			},
		},
	})
}
