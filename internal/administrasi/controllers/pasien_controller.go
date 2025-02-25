package controllers

import (
	"encoding/json"
	"strconv"

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

	// Validasi minimal
	if req.Nik == "" || req.IDPoli == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Nik and id_poli are required",
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

	// Buat objek Pasien untuk update (tidak mengubah NIK)
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

	// Panggil fungsi service untuk update data pasien dan registrasi kunjungan tambahan.
	idKunjungan, nomorAntrian, err := pc.Service.UpdatePasienAndRegisterKunjungan(p, req.IDPoli)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to register kunjungan: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Kunjungan registered successfully",
		"data": map[string]interface{}{
			"id_kunjungan":  idKunjungan,
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


// TundaPasienHandler menangani endpoint untuk menunda pasien berdasarkan id_antrian.
// Contoh URL: PUT http://localhost:8080/api/kunjungan/tunda?id_antrian=5
func (pc *PasienController) TundaPasienHandler(w http.ResponseWriter, r *http.Request) {
	idAntrianStr := r.URL.Query().Get("id_antrian")
	if idAntrianStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_antrian parameter is required",
			"data":    nil,
		})
		return
	}
	idAntrian, err := strconv.Atoi(idAntrianStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_antrian must be a number",
			"data":    nil,
		})
		return
	}

	err = pc.Service.TundaPasien(idAntrian)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to tunda pasien: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Pasien successfully tunda",
		"data":    nil,
	})
}


// RescheduleAntrianHandler menangani endpoint untuk mereschedule nomor antrian.
// Contoh URL: PUT http://localhost:8080/api/kunjungan/reschedule?id_antrian=5&id_poli=2
func (pc *PasienController) RescheduleAntrianHandler(w http.ResponseWriter, r *http.Request) {
	idAntrianStr := r.URL.Query().Get("id_antrian")
	idPoliStr := r.URL.Query().Get("id_poli")
	if idAntrianStr == "" || idPoliStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_antrian and id_poli parameters are required",
			"data":    nil,
		})
		return
	}
	idAntrian, err := strconv.Atoi(idAntrianStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_antrian must be a number",
			"data":    nil,
		})
		return
	}
	idPoli, err := strconv.Atoi(idPoliStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli must be a number",
			"data":    nil,
		})
		return
	}
	
	newPriority, err := pc.Service.RescheduleAntrianPriority(idAntrian, idPoli)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to reschedule antrian: " + err.Error(),
			"data":    nil,
		})
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Antrian rescheduled successfully",
		"data": map[string]interface{}{
			"new_priority_order": newPriority,
		},
	})
}

// GetAntrianTodayHandler menangani endpoint GET untuk mengambil data antrian hari ini dengan filter opsional.
func (pc *PasienController) GetAntrianTodayHandler(w http.ResponseWriter, r *http.Request) {
	// Ambil query parameter status, misalnya ?status=Menunggu
	statusFilter := r.URL.Query().Get("status")

	list, err := pc.Service.GetAntrianToday(statusFilter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve antrian: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Antrian retrieved successfully",
		"data":    list,
	})
}

func (pc *PasienController) GetAllStatusAntrianHandler(w http.ResponseWriter, r *http.Request) {
	list, err := pc.Service.GetAllStatusAntrian() // Pastikan Service di sini adalah *PendaftaranService
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve status antrian: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Status antrian retrieved successfully",
		"data":    list,
	})
}
