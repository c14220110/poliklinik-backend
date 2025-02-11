package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
)

type ExtendedPasienRequest struct {
	IDPoli       int    `json:"id_poli"`
	Nama         string `json:"nama"`
	JenisKelamin string `json:"jenis_kelamin"`
	TempatLahir  string `json:"tempat_lahir"`
	TanggalLahir string `json:"tanggal_lahir"`
	Nik          string `json:"nik"`
	NoTelp       string `json:"no_telp"`
	Alamat       string `json:"alamat"`
	Kelurahan    string `json:"kelurahan"`
	Kecamatan    string `json:"kecamatan"`
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

	// Parse tanggal_lahir
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

	// Buat objek Pasien (sesuai dengan struktur tabel)
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

	// Ambil operatorID dari token JWT. (Di sini diset sebagai dummy; seharusnya diambil dari context JWT)
	operatorID := 1

	patientID, nomorAntrian, err := pc.Service.RegisterPasienWithRekamMedisAndAntrian(p, req.IDPoli, operatorID)
	if err != nil {
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