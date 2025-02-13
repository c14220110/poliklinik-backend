package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
)

type AdministrasiController struct {
	Service *services.AdministrasiService
}

func NewAdministrasiController(service *services.AdministrasiService) *AdministrasiController {
	return &AdministrasiController{Service: service}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (ac *AdministrasiController) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload",
			"data":    nil,
		})
		return
	}

	admin, err := ac.Service.AuthenticateAdmin(req.Username, req.Password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid username or password",
			"data":    nil,
		})
		return
	}

	// Gunakan fungsi JWT terpadu: GenerateJWTToken
	// Konversi admin.ID_Admin ke string dan set id_poli = 0 karena admin tidak terkait dengan poli.
	token, err := utils.GenerateJWTToken(
		strconv.Itoa(admin.ID_Admin),
		"Administrasi",
		[]map[string]interface{}{
			{"privilege": "pendaftaran"},
			{"privilege": "billing"},
			{"privilege": "cetak_data"},
			{"privilege": "cetak_label"},
			{"privilege": "cetak_gelang"},
		},
		0,
		admin.Username,
	)
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
			"id":       admin.ID_Admin,
			"nama":     admin.Nama,
			"username": admin.Username,
			"token":    token,
		},
	})
}
