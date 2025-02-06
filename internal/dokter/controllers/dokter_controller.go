package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/dokter/models"
	"github.com/c14220110/poliklinik-backend/internal/dokter/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/golang-jwt/jwt/v4"
)

type DokterController struct {
	Service *services.DokterService
}

func NewDokterController(service *services.DokterService) *DokterController {
	return &DokterController{Service: service}
}

// LoginDokterRequest adalah struktur request untuk login dokter.
type LoginDokterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IDPoli   int    `json:"id_poli"` // Dokter memilih poliklinik yang ingin diakses
}

// CreateDokter registers a new dokter.
func (dc *DokterController) CreateDokter(w http.ResponseWriter, r *http.Request) {
	// Contoh implementasi register dokter
	var req struct {
		Nama         string `json:"nama"`
		Username     string `json:"username"`
		Password     string `json:"password"`
		Spesialisasi string `json:"spesialisasi"`
	}
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
			"message": "All fields are required",
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

	// Autentikasi dan validasi shift aktif
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

	// Generate token dengan claim tambahan "id_poli" dari shift aktif
	token, err := utils.GenerateTokenWithClaims(dokter.ID_Dokter, dokter.Username, map[string]interface{}{
		"id_poli": shift.ID_Poli,
	})
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
			"id_poli":      shift.ID_Poli, // dikirim sebagai verifikasi
			"token":        token,
			"shift": map[string]interface{}{
				"id_shift":    shift.ID_Shift,
				"jam_mulai":   shift.Jam_Mulai,
				"jam_selesai": shift.Jam_Selesai,
			},
		},
	})
}

// ListAntrian mengembalikan daftar antrian pasien yang statusnya 0 untuk poli yang sesuai dengan token.
func (dc *DokterController) ListAntrian(w http.ResponseWriter, r *http.Request) {
	// Ambil token dari header Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Authorization header missing",
			"data":    nil,
		})
		return
	}
	// Asumsikan format "Bearer <token>"
	var tokenString string
	_, err := fmt.Sscanf(authHeader, "Bearer %s", &tokenString)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid authorization header format",
			"data":    nil,
		})
		return
	}

	// Decode token
	token, err := utils.ValidateToken(tokenString)
	if err != nil || !token.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid token",
			"data":    nil,
		})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid token claims",
			"data":    nil,
		})
		return
	}

	// Ambil id_poli dari claims
	idPoliFloat, ok := claims["id_poli"].(float64)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "id_poli not found in token",
			"data":    nil,
		})
		return
	}
	idPoli := int(idPoliFloat)

	// Panggil service untuk mendapatkan antrian berdasarkan id_poli
	data, err := dc.Service.GetListAntrianByPoli(idPoli)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve antrian data: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Antrian data retrieved successfully",
		"data":    data,
	})
}
