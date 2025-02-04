package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
)

// PasienController menangani endpoint pendaftaran pasien.
type PasienController struct {
	Service *services.PendaftaranService
}

func NewPasienController(service *services.PendaftaranService) *PasienController {
	return &PasienController{Service: service}
}

// ExtendedPasienRequest digunakan untuk menangkap data pendaftaran pasien yang dikirim dari frontend,
// dengan tanggal_lahir dalam bentuk string.
type ExtendedPasienRequest struct {
	Nama         string `json:"nama"`
	TanggalLahir string `json:"tanggal_lahir"`  // diharapkan format "2006-01-02"
	JenisKelamin string `json:"jenis_kelamin"`
	TempatLahir  string `json:"tempat_lahir"`
	Nik          string `json:"nik"`
	Kelurahan    string `json:"kelurahan"`
	Kecamatan    string `json:"kecamatan"`
	Alamat       string `json:"alamat"`
	NoTelp       string `json:"no_telp"`
	IDPoli       int    `json:"id_poli"`
}

// RegisterPasien menerima data pendaftaran pasien baru dan membuat entri antrian dalam satu transaksi.
func (pc *PasienController) RegisterPasien(w http.ResponseWriter, r *http.Request) {
	var req ExtendedPasienRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload",
			"data":    nil,
		})
		return
	}

	// Validasi field wajib
	if req.Nama == "" || req.IDPoli == 0 || req.TanggalLahir == "" || req.Nik == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Nama, tanggal_lahir, NIK, dan id_poli harus diisi",
			"data":    nil,
		})
		return
	}

	// Parse tanggal_lahir dari string ke time.Time
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

	// Buat objek Pasien berdasarkan request
	p := models.Pasien{
		Nama:         req.Nama,
		TanggalLahir: parsedDate,
		JenisKelamin: req.JenisKelamin,
		TempatLahir:  req.TempatLahir,
		Nik:          req.Nik,
		Kelurahan:    req.Kelurahan,
		Kecamatan:    req.Kecamatan,
		Alamat:       req.Alamat,
		NoTelp:       req.NoTelp,
		TanggalRegistrasi: time.Now(),
	}

	// Buat data antrian
	var a models.Antrian
	a.IDPoli = req.IDPoli

	patientID, nomorAntrian, err := pc.Service.RegisterPasienWithAntrian(p, a)
	if err != nil {
		// Jika error adalah "NIK sudah terdaftar", kirimkan status 409 (Conflict)
		if err.Error() == "NIK sudah terdaftar" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusConflict,
				"message": "NIK sudah terdaftar dalam sistem",
				"data":    nil,
			})
			return
		}

		// Jika error lain, kembalikan status 500
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Gagal mendaftarkan pasien: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Response sukses
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Pasien berhasil didaftarkan",
		"data": map[string]interface{}{
			"id":            patientID,
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