package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	"github.com/c14220110/poliklinik-backend/ws"
	"github.com/labstack/echo/v4"
)

type ExtendedPasienRequest struct {
	Nama              string `json:"nama"`
	JenisKelamin      string `json:"jenis_kelamin"`
	TempatLahir       string `json:"tempat_lahir"`
	TanggalLahir      string `json:"tanggal_lahir"`
	Nik               string `json:"nik"`
	NoTelp            string `json:"no_telp"`
	Alamat            string `json:"alamat"`
	Kelurahan         string `json:"kelurahan"`
	Kecamatan         string `json:"kecamatan"`
	KotaTempatTinggal string `json:"kota_tempat_tinggal"`
	IDPoli            int    `json:"id_poli"`
	KeluhanUtama      string `json:"keluhan_utama"`
}

type PasienController struct {
	Service *services.PendaftaranService
}

func NewPasienController(service *services.PendaftaranService) *PasienController {
	return &PasienController{Service: service}
}


func (pc *PasienController) RegisterPasien(c echo.Context) error {
    var req ExtendedPasienRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "Invalid request payload: " + err.Error(),
            "data":    nil,
        })
    }

    // Validasi field wajib
    if req.Nama == "" || req.TanggalLahir == "" || req.Nik == "" || req.IDPoli == 0 {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "Nama, tanggal_lahir, nik, dan id_poli harus diisi",
            "data":    nil,
        })
    }

    // Parse tanggal lahir
    parsedDate, err := time.Parse("2006-01-02", req.TanggalLahir)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "Format tanggal_lahir tidak valid. Gunakan format YYYY-MM-DD",
            "data":    nil,
        })
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
        KotaTinggal:  req.KotaTempatTinggal,
    }

    // Operator yang menginput (dummy ID 1, idealnya dari JWT)
    operatorID := 1

   patientID, idAntrian, nomorAntrian, idRM, idStatus, namaPoli, err := pc.Service.RegisterPasienWithKunjungan(p, req.IDPoli, operatorID, req.KeluhanUtama)
if err != nil {
    if err.Error() == "NIK sudah terdaftar" {
        return c.JSON(http.StatusConflict, map[string]interface{}{
            "status":  http.StatusConflict,
            "message": "NIK sudah terdaftar",
            "data":    nil,
        })
    }
    return c.JSON(http.StatusInternalServerError, map[string]interface{}{
        "status":  http.StatusInternalServerError,
        "message": "Gagal mendaftarkan pasien: " + err.Error(),
        "data":    nil,
    })
}

// Siapkan data broadcast dengan format yang diinginkan
broadcastData := map[string]interface{}{
    "id_antrian":    idAntrian,
    "id_pasien":     patientID,
    "id_poli":       req.IDPoli,
    "id_rm":         idRM,
    "id_status":     idStatus,
    "nama":          req.Nama,
    "nama_poli":     namaPoli,
    "nomor_antrian": nomorAntrian,
    "priority_order": nomorAntrian, // misal sama dengan nomor_antrian
    "status":        "Menunggu",      // sesuai contoh format broadcast
}

messageJSON, err := json.Marshal(broadcastData)
if err != nil {
    return c.JSON(http.StatusInternalServerError, map[string]interface{}{
        "status":  http.StatusInternalServerError,
        "message": "Gagal membuat pesan broadcast: " + err.Error(),
        "data":    nil,
    })
}

// Kirim pesan ke WebSocket
ws.HubInstance.Broadcast <- messageJSON

// Kembalikan response API ke client
return c.JSON(http.StatusOK, map[string]interface{}{
    "status":  http.StatusOK,
    "message": "Pasien berhasil didaftarkan",
    "data": map[string]interface{}{
        "id_pasien":     patientID,
        "id_antrian":    idAntrian,
        "nomor_antrian": nomorAntrian,
    },
})

}

func (pc *PasienController) UpdateKunjungan(c echo.Context) error {
	var req ExtendedPasienRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
	}

	// Validasi minimal
	if req.Nik == "" || req.IDPoli == 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Nik and id_poli are required",
			"data":    nil,
		})
	}

	parsedDate, err := time.Parse("2006-01-02", req.TanggalLahir)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid date format for tanggal_lahir. Use YYYY-MM-DD",
			"data":    nil,
		})
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
		KotaTinggal:  req.KotaTempatTinggal,
	}

	// Panggil service update yang sudah dimodifikasi agar mengembalikan data tambahan
	idPasien, idAntrian, nomorAntrian, idRM, idStatus, namaPoli, err := pc.Service.UpdatePasienAndRegisterKunjungan(p, req.IDPoli, req.KeluhanUtama)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to register kunjungan: " + err.Error(),
			"data":    nil,
		})
	}

	// Siapkan data broadcast dengan format JSON
	broadcastData := map[string]interface{}{
		"id_antrian":    idAntrian,
		"id_pasien":     idPasien,
		"id_poli":       req.IDPoli,
		"id_rm":         idRM,
		"id_status":     idStatus,
		"nama":          req.Nama,
		"nama_poli":     namaPoli,
		"nomor_antrian": nomorAntrian,
		"priority_order": nomorAntrian, // misalnya sama dengan nomor_antrian
		"status":        "Menunggu",     // sesuai dengan data di database
	}

	messageJSON, err := json.Marshal(broadcastData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to marshal broadcast message: " + err.Error(),
			"data":    nil,
		})
	}

	// Kirim pesan ke WebSocket
	ws.HubInstance.Broadcast <- messageJSON

	// Kembalikan response API ke client
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Kunjungan registered successfully",
		"data": map[string]interface{}{
			"id_pasien":     idPasien,
			"id_antrian":    idAntrian,
			"nomor_antrian": nomorAntrian,
		},
	})
}


func (pc *PasienController) GetAllPasienData(c echo.Context) error {
	// Ambil query parameter "nama", "page", dan "limit"
	namaFilter := c.QueryParam("nama")
	pageStr := c.QueryParam("page")
	limitStr := c.QueryParam("limit")

	// Set default pagination: page = 1, limit = 20 jika tidak disediakan
	page := 1
	limit := 20
	var err error
	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
	}
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			limit = 20
		}
	}

	list, err := pc.Service.GetAllPasienDataFiltered(namaFilter, page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve pasien data: " + err.Error(),
			"data":    nil,
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Data pasien retrieved successfully",
		"data":    list,
	})
}



func (pc *PasienController) TundaPasienHandler(c echo.Context) error {
	idAntrianStr := c.QueryParam("id_antrian")
	if idAntrianStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "parameter id_antrian wajib diisi",
			"data":    nil,
		})
	}
	idAntrian, err := strconv.Atoi(idAntrianStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_antrian harus berupa angka",
			"data":    nil,
		})
	}

	err = pc.Service.TundaPasien(idAntrian)
	if err != nil {
		if strings.Contains(err.Error(), "tidak ditemukan") {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"status":  http.StatusNotFound,
				"message": err.Error(),
				"data":    nil,
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Gagal menunda pasien: " + err.Error(),
			"data":    nil,
		})
	}

	// Setelah update berhasil, trigger event broadcast ke WebSocket.
    data := map[string]interface{}{
	"id_antrian": idAntrian,
	"status":     "Ditunda",
    }
    messageJSON, err := json.Marshal(data)
    if err != nil {
	    return c.JSON(http.StatusInternalServerError, map[string]interface{}{
		"status":  http.StatusInternalServerError,
		"message": "Gagal membuat pesan JSON: " + err.Error(),
		"data":    nil,
	})
    }
    ws.HubInstance.Broadcast <- messageJSON
    
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Pasien berhasil ditunda",
		"data":    nil,
	})
}


func (pc *PasienController) RescheduleAntrianHandler(c echo.Context) error {
    idAntrianStr := c.QueryParam("id_antrian")
    if idAntrianStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "parameter id_antrian wajib diisi",
            "data":    nil,
        })
    }
    idAntrian, err := strconv.Atoi(idAntrianStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_antrian harus berupa angka",
            "data":    nil,
        })
    }

    newPriority, err := pc.Service.RescheduleAntrianPriority(idAntrian)
    if err != nil {
        if strings.Contains(err.Error(), "tidak ditemukan") || strings.Contains(err.Error(), "tidak dalam status 'Ditunda'") {
            return c.JSON(http.StatusBadRequest, map[string]interface{}{
                "status":  http.StatusBadRequest,
                "message": err.Error(),
                "data":    nil,
            })
        }
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Gagal mereschedule antrian: " + err.Error(),
            "data":    nil,
        })
    }

    // Siapkan data broadcast untuk WebSocket
    broadcastData := map[string]interface{}{
        "id_antrian":         idAntrian,
        "new_priority_order": newPriority,
        "status":             "Menunggu",
    }

    messageJSON, err := json.Marshal(broadcastData)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Gagal membuat pesan broadcast: " + err.Error(),
            "data":    nil,
        })
    }

    // Kirim pesan ke WebSocket
    ws.HubInstance.Broadcast <- messageJSON

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Antrian berhasil di-reschedule",
        "data": map[string]interface{}{
            "new_priority_order": newPriority,
        },
    })
}


func (pc *PasienController) GetAntrianTodayHandler(c echo.Context) error {
    statusFilter := c.QueryParam("status")
    list, err := pc.Service.GetAntrianToday(statusFilter)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve antrian: " + err.Error(),
            "data":    nil,
        })
    }
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Antrian retrieved successfully",
        "data":    list,
    })
}

func (pc *PasienController) GetAllStatusAntrianHandler(c echo.Context) error {
    list, err := pc.Service.GetAllStatusAntrian()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve status antrian: " + err.Error(),
            "data":    nil,
        })
    }
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Status antrian retrieved successfully",
        "data":    list,
    })
}

func (pc *PasienController) BatalkanAntrianHandler(c echo.Context) error {
    // 1. Ambil query parameter id_antrian
    idAntrianStr := c.QueryParam("id_antrian")
    if idAntrianStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "parameter id_antrian wajib diisi",
            "data":    nil,
        })
    }

    // 2. Konversi ke integer
    idAntrian, err := strconv.Atoi(idAntrianStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_antrian harus berupa angka",
            "data":    nil,
        })
    }

    // 3. Panggil fungsi service untuk membatalkan antrian
    err = pc.Service.BatalkanAntrian(idAntrian)
    if err != nil {
        if strings.Contains(err.Error(), "tidak ditemukan") {
            return c.JSON(http.StatusNotFound, map[string]interface{}{
                "status":  http.StatusNotFound,
                "message": err.Error(),
                "data":    nil,
            })
        }
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "gagal membatalkan antrian: " + err.Error(),
            "data":    nil,
        })
    }

    // 4. Kirim broadcast ke WebSocket
    broadcastData := map[string]interface{}{
        "id_antrian": idAntrian,
        "status":     "Dibatalkan",
    }
    messageJSON, err := json.Marshal(broadcastData)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Gagal membuat pesan broadcast: " + err.Error(),
            "data":    nil,
        })
    }
    ws.HubInstance.Broadcast <- messageJSON

    // 5. Respons sukses
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "antrian berhasil dibatalkan",
        "data":    nil,
    })
}
