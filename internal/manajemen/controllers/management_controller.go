package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
)

type ManagementController struct {
	Service *services.ManagementService
}

func NewManagementController(service *services.ManagementService) *ManagementController {
	return &ManagementController{Service: service}
}

type LoginManagementRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (mc *ManagementController) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginManagementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload",
			"data":    nil,
		})
		return
	}
	if req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Username and Password are required",
			"data":    nil,
		})
		return
	}

	// Autentikasi manajemen melalui service
	m, err := mc.Service.AuthenticateManagement(req.Username, req.Password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid username or password",
			"data":    nil,
		})
		return
	}

	// Siapkan privileges untuk management
	extraClaims := []map[string]interface{}{
		{"privilege": "manage_dashboard"},
		{"privilege": "manage_poli"},
		{"privilege": "manage_shift"},
		{"privilege": "manage_user"},
		{"privilege": "manage_role_privilege"},
	}

	// Gunakan GenerateJWTToken dari utils. Karena management tidak terkait poli, set id_poli=0.
	token, err := utils.GenerateJWTToken(
		strconv.Itoa(m.ID_Management),
		"Manajemen",
		extraClaims,
		0,
		m.Username,
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

	// Kirim response standar
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Login successful",
		"data": map[string]interface{}{
			"id":       m.ID_Management,
			"nama":     m.Nama,
			"username": m.Username,
			"token":    token,
		},
	})
}
