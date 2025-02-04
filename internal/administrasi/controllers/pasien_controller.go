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

// ExtendedPasien digunakan untuk menangkap data pendaftaran pasien yang dikirim dari frontend,
// termasuk ID_Poli yang dipilih oleh admin.
type ExtendedPasien struct {
	models.Pasien
	IDPoli int `json:"id_poli"`
}

// RegisterPasien menerima data pendaftaran pasien baru dan membuat entri antrian dalam satu transaksi.
func (pc *PasienController) RegisterPasien(w http.ResponseWriter, r *http.Request) {
	var ep ExtendedPasien
	if err := json.NewDecoder(r.Body).Decode(&ep); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload",
			"data":    nil,
		})
		return
	}

	// Validasi field wajib
	if ep.Nama == "" || ep.PoliTujuan == "" || ep.IDPoli == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Nama, Poli Tujuan, dan id_poli harus diisi",
			"data":    nil,
		})
		return
	}

	// Set Tanggal_Registrasi
	ep.TanggalRegistrasi = time.Now()

	// Buat data antrian; nomor antrian dan status akan dihitung di service.
	var a models.Antrian
	a.IDPoli = ep.IDPoli

	patientID, nomorAntrian, err := pc.Service.RegisterPasienWithAntrian(ep.Pasien, a)
	if err != nil {
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
