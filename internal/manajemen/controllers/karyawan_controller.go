package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
)

type AddKaryawanRequest struct {
	NIK          string `json:"nik"`
	Nama         string `json:"nama"`
	TanggalLahir string `json:"tanggal_lahir"`
	Alamat       string `json:"alamat"`
	NoTelp       string `json:"no_telp"`
	Role         string `json:"role"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}
type UpdateKaryawanRequest struct {
	IDKaryawan   int64  `json:"id_karyawan"`
	NIK          string `json:"nik"`
	Nama         string `json:"nama"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	TanggalLahir string `json:"tanggal_lahir"`
	Alamat       string `json:"alamat"`
	NoTelp       string `json:"no_telp"`
	Role         string `json:"role"`
}

type KaryawanController struct {
	Service *services.ManagementService
}

func NewKaryawanController(service *services.ManagementService) *KaryawanController {
	return &KaryawanController{Service: service}
}

func (kc *KaryawanController) AddKaryawan(w http.ResponseWriter, r *http.Request) {
	var req AddKaryawanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Parse tanggal lahir
	parsedDate, err := time.Parse("2006-01-02", req.TanggalLahir)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid date format for tanggal_lahir. Use YYYY-MM-DD",
			"data":    nil,
		})
		return
	}

	// Buat objek Karyawan
	karyawan := models.Karyawan{
		NIK:          req.NIK,
		Nama:         req.Nama,
		TanggalLahir: parsedDate,
		Alamat:       req.Alamat,
		NoTelp:       req.NoTelp,
		Username:     req.Username,
		Password:     req.Password,
	}

	// Ambil klaim JWT dari context untuk mendapatkan id_management dan username management yang sedang login
	claims, ok := r.Context().Value(middlewares.ContextKeyClaims).(*utils.Claims)
	if !ok || claims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
		return
	}

	idManagement, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idManagement <= 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid management ID in token",
			"data":    nil,
		})
		return
	}

	// Panggil service untuk menambahkan karyawan
	idKaryawan, err := kc.Service.AddKaryawan(karyawan, req.Role, idManagement, claims.Username, claims.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to add karyawan: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Karyawan added successfully",
		"data": map[string]interface{}{
			"id_karyawan": idKaryawan,
		},
	})
}



// GetKaryawanListHandler mengembalikan daftar karyawan dengan field: id_karyawan, nama, nik, tanggal_lahir, role, tahun_kerja.
func (kc *KaryawanController) GetKaryawanListHandler(w http.ResponseWriter, r *http.Request) {
	karyawanList, err := kc.Service.GetKaryawanList()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve karyawan list: " + err.Error(),
			"data":    nil,
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Karyawan list retrieved successfully",
		"data":    karyawanList,
	})
}

func (kc *KaryawanController) UpdateKaryawanHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateKaryawanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Validasi minimal field
	if req.IDKaryawan == 0 || req.NIK == "" || req.Nama == "" || req.Username == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_karyawan, nik, nama, username, and password are required",
			"data":    nil,
		})
		return
	}

	// Parse tanggal_lahir dengan format "YYYY-MM-DD"
	parsedDate, err := time.Parse("2006-01-02", req.TanggalLahir)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid date format for tanggal_lahir. Use YYYY-MM-DD",
			"data":    nil,
		})
		return
	}

	// Buat objek Karyawan untuk update; NIK tidak boleh diubah
	karyawan := models.Karyawan{
		IDKaryawan:   req.IDKaryawan,
		NIK:          req.NIK,
		Nama:         req.Nama,
		Username:     req.Username,
		Password:     req.Password,
		TanggalLahir: parsedDate,
		Alamat:       req.Alamat,
		NoTelp:       req.NoTelp,
	}

	// Ambil klaim JWT dari context untuk mendapatkan informasi management yang sedang login
	claims, ok := r.Context().Value(middlewares.ContextKeyClaims).(*utils.Claims)
	if !ok || claims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
		return
	}

	// Ambil id_management dari token (di sini menggunakan klaim id_karyawan sebagai id_management)
	idManagement, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idManagement <= 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid management ID in token",
			"data":    nil,
		})
		return
	}

	// Panggil service untuk update karyawan; gunakan claims.Username sebagai updated_by
	updatedID, err := kc.Service.UpdateKaryawan(karyawan, req.Role, idManagement, claims.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to update karyawan: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Kembalikan respons sukses
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Karyawan updated successfully",
		"data": map[string]interface{}{
			"id_karyawan": updatedID,
		},
	})
}