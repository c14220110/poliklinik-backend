package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/screening/models"
	"github.com/c14220110/poliklinik-backend/internal/screening/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
)

type SusterController struct {
	Service *services.SusterService
}

func NewSusterController(service *services.SusterService) *SusterController {
	return &SusterController{Service: service}
}

// CreateSusterRequest adalah struktur request untuk pembuatan suster.
type CreateSusterRequest struct {
	Nama     string `json:"nama"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateSuster mendaftarkan suster baru.
func (sc *SusterController) CreateSuster(w http.ResponseWriter, r *http.Request) {
	var req CreateSusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload",
			"data":    nil,
		})
		return
	}
	if req.Nama == "" || req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Nama, Username, and Password are required",
			"data":    nil,
		})
		return
	}

	suster := models.Suster{
		Nama:     req.Nama,
		Username: req.Username,
		Password: req.Password,
	}
	id, err := sc.Service.CreateSuster(suster)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to create suster: " + err.Error(),
			"data":    nil,
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Suster created successfully",
		"data": map[string]interface{}{
			"id": id,
		},
	})
}

// LoginSusterRequest adalah struktur request untuk login suster.
type LoginSusterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IDPoli   int    `json:"id_poli"` // Poli yang dipilih oleh suster
}

// LoginSuster menangani login suster dengan validasi shift aktif.
func (sc *SusterController) LoginSuster(w http.ResponseWriter, r *http.Request) {
	var req LoginSusterRequest
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
			"message": "Username, Password, and id_poli are required",
			"data":    nil,
		})
		return
	}

	suster, shift, err := sc.Service.AuthenticateSuster(req.Username, req.Password, req.IDPoli)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	// Generate JWT token untuk suster
	token, err := utils.GenerateToken(suster.ID_Suster, suster.Username)
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
			"id":       suster.ID_Suster,
			"nama":     suster.Nama,
			"username": suster.Username,
			"id_poli":  shift.ID_Poli, // id_poli dari shift aktif
			"token":    token,
			"shift": map[string]interface{}{
				"id_shift":    shift.ID_Shift,
				"jam_mulai":   shift.Jam_Mulai,
				"jam_selesai": shift.Jam_Selesai,
			},
		},
	})
}
