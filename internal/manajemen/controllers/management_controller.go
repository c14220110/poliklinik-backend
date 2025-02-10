package controllers

import (
	"encoding/json"
	"net/http"

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

	// Autentikasi manajemen
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

	// Sisipkan klaim tambahan ke token JWT
	extraClaims := map[string]interface{}{
		"role":       "Manajemen",
		"privileges": []string{"manage_dashboard", "manage_poli", "manage_shift", "manage_user", "manage_role_privilege"},
	}

	token, err := utils.GenerateTokenWithClaims(m.ID_Management, m.Username, extraClaims)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to generate token: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Response standar: status, message, data
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