package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
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

// **LOGIN FUNCTION WITH STANDARDIZED RESPONSE**
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

	admin, err := ac.Service.Authenticate(req.Username, req.Password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid username or password",
			"data":    nil,
		})
		return
	}

	// Generate JWT token setelah autentikasi berhasil.
	token, err := utils.GenerateToken(admin.ID, admin.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to generate token",
			"data":    nil,
		})
		return
	}

	// **Response sukses dengan format standar**
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Login successful",
		"data": map[string]interface{}{
			"id":       admin.ID,
			"nama":     admin.Nama,
			"username": admin.Username,
			"token":    token,
		},
	})
}


// **LOGOUT FUNCTION WITH STANDARDIZED RESPONSE**
func (ac *AdministrasiController) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Logout successful",
		"data":    nil,
	})
}


// CreateAdminRequest merupakan struktur request untuk pembuatan admin baru.
type CreateAdminRequest struct {
	Nama     string `json:"nama"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateAdmin menerima request untuk membuat admin baru dan menghash password-nya.
func (ac *AdministrasiController) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	var req CreateAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validasi field yang wajib diisi.
	if req.Nama == "" || req.Username == "" || req.Password == "" {
		http.Error(w, "Nama, Username, and Password are required", http.StatusBadRequest)
		return
	}

	newAdmin := models.Administrasi{
		Nama:     req.Nama,
		Username: req.Username,
		Password: req.Password,
	}

	id, err := ac.Service.CreateAdmin(newAdmin)
	if err != nil {
		http.Error(w, "Failed to create admin: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":      id,
		"message": "Admin created successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
