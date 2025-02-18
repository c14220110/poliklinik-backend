package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
)

type KaryawanController struct {
	Service *services.ManagementService
}

func NewKaryawanController(service *services.ManagementService) *KaryawanController {
	return &KaryawanController{Service: service}
}

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

	// Ambil ID_Management dari token JWT
	idManagement := 1 // Dummy value for now, replace with actual user_id from token

	// Panggil service untuk tambah karyawan dan role
	idKaryawan, err := kc.Service.AddKaryawan(karyawan, req.Role, idManagement)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to add karyawan: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Response sukses
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