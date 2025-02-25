package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
)

type RoleController struct {
	Service *services.RoleService
}

func NewRoleController(service *services.RoleService) *RoleController {
	return &RoleController{Service: service}
}

// AddRoleHandler menangani endpoint POST untuk menambah Role.
func (rc *RoleController) AddRoleHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NamaRole string `json:"nama_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}
	if req.NamaRole == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "nama_role is required",
			"data":    nil,
		})
		return
	}

	id, err := rc.Service.AddRole(req.NamaRole)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Role added successfully",
		"data":    map[string]interface{}{"id_role": id},
	})
}

// UpdateRoleHandler menangani endpoint PUT untuk memperbarui Role berdasarkan query parameter id_role.
func (rc *RoleController) UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	idRoleStr := r.URL.Query().Get("id_role")
	if idRoleStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_role parameter is required",
			"data":    nil,
		})
		return
	}
	idRole, err := strconv.Atoi(idRoleStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_role must be a number",
			"data":    nil,
		})
		return
	}

	var req struct {
		NamaRole string `json:"nama_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}
	if req.NamaRole == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "nama_role is required",
			"data":    nil,
		})
		return
	}

	err = rc.Service.UpdateRole(idRole, req.NamaRole)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Role updated successfully",
		"data":    map[string]interface{}{"id_role": idRole},
	})
}

// SoftDeleteRoleHandler menangani endpoint PUT untuk soft delete Role berdasarkan query parameter id_role.
func (rc *RoleController) SoftDeleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	// Pastikan metode PUT
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusMethodNotAllowed,
			"message": "Method not allowed",
			"data":    nil,
		})
		return
	}

	idRoleStr := r.URL.Query().Get("id_role")
	if idRoleStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_role parameter is required",
			"data":    nil,
		})
		return
	}
	idRole, err := strconv.Atoi(idRoleStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_role must be a number",
			"data":    nil,
		})
		return
	}

	err = rc.Service.SoftDeleteRole(idRole)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Role soft-deleted successfully",
		"data":    map[string]interface{}{"id_role": idRole},
	})
}

// GetRoleListHandler menangani endpoint GET untuk mengambil daftar role dengan filter opsional.
func (rc *RoleController) GetRoleListHandler(w http.ResponseWriter, r *http.Request) {
	// Ambil query parameter "status" (opsional)
	statusFilter := r.URL.Query().Get("status")

	roles, err := rc.Service.GetRoleListFiltered(statusFilter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve role list: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Role list retrieved successfully",
		"data":    roles,
	})
}

// ActivateRoleHandler mengubah deleted_at menjadi NULL untuk role tertentu.
func (rc *RoleController) ActivateRoleHandler(w http.ResponseWriter, r *http.Request) {
	// Pastikan metode PUT
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusMethodNotAllowed,
			"message": "Method not allowed",
			"data":    nil,
		})
		return
	}

	idRoleStr := r.URL.Query().Get("id_role")
	if idRoleStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_role parameter is required",
			"data":    nil,
		})
		return
	}
	idRole, err := strconv.Atoi(idRoleStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_role must be a number",
			"data":    nil,
		})
		return
	}

	err = rc.Service.ActivateRole(idRole)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to activate role: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Role activated successfully",
		"data":    map[string]interface{}{"id_role": idRole},
	})
}