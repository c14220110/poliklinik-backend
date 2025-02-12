package controllers

import (
	"encoding/json"
	//"fmt"
	"net/http"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
)

type ExtendedPasienRequest struct {
	Nama         string `json:"nama"`
	JenisKelamin string `json:"jenis_kelamin"`
	TempatLahir  string `json:"tempat_lahir"`
	TanggalLahir string `json:"tanggal_lahir"`
	Nik          string `json:"nik"`
	NoTelp       string `json:"no_telp"`
	Alamat       string `json:"alamat"`
	Kelurahan    string `json:"kelurahan"`
	Kecamatan    string `json:"kecamatan"`
	IDPoli       int    `json:"id_poli"`
}

type PasienController struct {
	Service *services.PendaftaranService
}

func NewPasienController(service *services.PendaftaranService) *PasienController {
	return &PasienController{Service: service}
}

func (pc *PasienController) RegisterPasien(w http.ResponseWriter, r *http.Request) {
	var req ExtendedPasienRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Validasi field wajib
	if req.Nama == "" || req.TanggalLahir == "" || req.Nik == "" || req.IDPoli == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Nama, tanggal_lahir, nik, dan id_poli harus diisi",
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
			"message": "Format tanggal_lahir tidak valid. Gunakan format YYYY-MM-DD",
			"data":    nil,
		})
		return
	}

	// Buat objek Pasien
	p := models.Pasien{
		Nama:         req.Nama,
		TanggalLahir: parsedDate,
		JenisKelamin: req.JenisKelamin,
		TempatLahir:  req.TempatLahir,
		NIK:          req.Nik,
		Kelurahan:    req.Kelurahan,
		Kecamatan:    req.Kecamatan,
		Alamat:       req.Alamat,
		NoTelp:       req.NoTelp,
	}

	// Operator yang menginput; idealnya didapat dari token JWT, di sini dummy (misal, 1)
	operatorID := 1

	patientID, nomorAntrian, err := pc.Service.RegisterPasienWithKunjungan(p, req.IDPoli, operatorID)
	if err != nil {
		// Jika error adalah "NIK sudah terdaftar", kembalikan status Conflict (409)
		if err.Error() == "NIK sudah terdaftar" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusConflict,
				"message": "NIK sudah terdaftar",
				"data":    nil,
			})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Gagal mendaftarkan pasien: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Pasien berhasil didaftarkan",
		"data": map[string]interface{}{
			"id_pasien":     patientID,
			"nomor_antrian": nomorAntrian,
		},
	})
}



// ListPasien mengembalikan daftar pasien beserta informasi antrian dan rekam medis.
func (pc *PasienController) ListPasien(w http.ResponseWriter, r *http.Request) {
	list, err := pc.Service.GetListPasien()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Gagal mengambil data pasien: " + err.Error(),
			"data":    nil,
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Data pasien berhasil diambil",
		"data":    list,
	})
}

// UpdateKunjungan mencatat kunjungan tambahan untuk pasien yang sudah terdaftar berdasarkan NIK.
func (pc *PasienController) UpdateKunjungan(w http.ResponseWriter, r *http.Request) {
	var req ExtendedPasienRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Pastikan NIK dan id_poli ada
	if req.Nik == "" || req.IDPoli == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "NIK dan id_poli harus diisi",
			"data":    nil,
		})
		return
	}

	// Parse tanggal_lahir jika perlu (jika ingin mengupdate data pasien, bisa disertakan)
	if req.TanggalLahir != "" {
		_, err := time.Parse("2006-01-02", req.TanggalLahir)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusBadRequest,
				"message": "Format tanggal_lahir tidak valid. Gunakan format YYYY-MM-DD",
				"data":    nil,
			})
			return
		}
	}

	// Buat objek pasien untuk update (hanya field yang ingin diupdate, misalnya Nama dan No_Telp)
	updatedPatient := models.Pasien{
		Nama:   req.Nama,
		// Jika NoTelp kosong, bisa tetap dikosongkan atau tidak diupdate.
		NoTelp: req.NoTelp,
	}

	patientID, nomorAntrian, err := pc.Service.UpdateKunjunganPasien(req.Nik, updatedPatient, req.IDPoli)
	if err != nil {
		// Jika error menyatakan pasien tidak ditemukan, kembalikan 404
		if err.Error() == "Pasien dengan NIK "+req.Nik+" tidak ditemukan" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusNotFound,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Gagal mencatat kunjungan: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Kunjungan pasien berhasil dicatat",
		"data": map[string]interface{}{
			"id_pasien":     patientID,
			"nomor_antrian": nomorAntrian,
		},
	})
}

func (pc *PasienController) GetAllPasienData(w http.ResponseWriter, r *http.Request) {
	list, err := pc.Service.GetAllPasienData()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve pasien data: " + err.Error(),
			"data":    nil,
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Data pasien retrieved successfully",
		"data":    list,
	})
}
