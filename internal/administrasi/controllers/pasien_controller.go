package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
)

type PasienController struct {
	Service *services.PendaftaranService
}

func NewPasienController(service *services.PendaftaranService) *PasienController {
	return &PasienController{Service: service}
}

func (pc *PasienController) RegisterPasien(w http.ResponseWriter, r *http.Request) {
	var p models.Pasien
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validasi sederhana
	if p.Nama == "" || p.PoliTujuan == "" {
		http.Error(w, "Nama dan Poli Tujuan harus diisi", http.StatusBadRequest)
		return
	}

	// Set tanggal registrasi saat ini
	p.TanggalRegistrasi = time.Now()

	id, err := pc.Service.DaftarPasien(p)
	if err != nil {
		http.Error(w, "Gagal mendaftarkan pasien", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":      id,
		"message": "Pasien berhasil didaftarkan",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (pc *PasienController) ListPasien(w http.ResponseWriter, r *http.Request) {
	list, err := pc.Service.GetListPasien()
	if err != nil {
		http.Error(w, "Failed to retrieve patients", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
