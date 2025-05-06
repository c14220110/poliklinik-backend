package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	common "github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	jwtUtils "github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/c14220110/poliklinik-backend/ws"
	"github.com/labstack/echo/v4"
)

// ExtendedPasienRequest defines payload untuk pendaftaran pasien


type PasienController struct {
	Service *services.PendaftaranService
}

func NewPasienController(service *services.PendaftaranService) *PasienController {
	return &PasienController{Service: service}
}

// RegisterPasien mendaftarkan pasien baru + kunjungan + broadcast WS
func (pc *PasienController) RegisterPasien(c echo.Context) error {
	var req models.ExtendedPasienRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
	}
	// Validasi field wajib
	if req.Nama == "" || req.TanggalLahir == "" || req.Nik == "" || req.IDPoli == 0 || req.Agama == "" || req.StatusPerkawinan == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Nama, tanggal_lahir, nik, id_poli, agama, dan status_perkawinan harus diisi",
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
	// Ambil operatorID dari JWT
	claims := c.Get(string(common.ContextKeyClaims)).(*jwtUtils.Claims)
	operatorID, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid operator ID in token",
			"data":    nil,
		})
	}
	// Cari ID agama berdasarkan nama agama
	var idAgama int
	err = pc.Service.DB.QueryRow("SELECT id_agama FROM Agama WHERE nama = ?", req.Agama).Scan(&idAgama)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  http.StatusBadRequest,
				"message": "Agama tidak ditemukan",
				"data":    nil,
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Gagal mencari agama: " + err.Error(),
			"data":    nil,
		})
	}
	// Konversi status perkawinan
	var statusPerkawinan int
	if strings.ToLower(req.StatusPerkawinan) == "sudah kawin" {
		statusPerkawinan = 1
	} else if strings.ToLower(req.StatusPerkawinan) == "belum kawin" {
		statusPerkawinan = 0
	} else {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Status perkawinan tidak valid. Harus 'sudah kawin' atau 'belum kawin'",
			"data":    nil,
		})
	}
	// Bangun objek Pasien
	p := models.Pasien{
		Nama:             req.Nama,
		TanggalLahir:     parsedDate,
		JenisKelamin:     req.JenisKelamin,
		TempatLahir:      req.TempatLahir,
		NIK:              req.Nik,
		Kelurahan:        req.Kelurahan,
		Kecamatan:        req.Kecamatan,
		KotaTinggal:      req.KotaTempatTinggal,
		Alamat:           req.Alamat,
		NoTelp:           req.NoTelp,
		IDAgama:          idAgama,
		StatusPerkawinan: statusPerkawinan,
		Pekerjaan:        req.Pekerjaan,
	}
	// Panggil service
	patientID, idAntrian, nomorAntrian, idRM, idStatus, namaPoli, err :=
		pc.Service.RegisterPasienWithKunjungan(p, req.IDPoli, operatorID, req.KeluhanUtama, req.PenanggungJawab)
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
	// Siapkan payload WS
	inner := map[string]interface{}{
		"id_antrian":     idAntrian,
		"id_pasien":      patientID,
		"id_poli":        req.IDPoli,
		"id_rm":          idRM,
		"id_status":      idStatus,
		"nama":           req.Nama,
		"nama_poli":      namaPoli,
		"nomor_antrian":  nomorAntrian,
		"priority_order": nomorAntrian,
		"status":         "Menunggu",
	}
	wrapper := map[string]interface{}{
		"type": "antrian_update",
		"data": inner,
	}
	msg, _ := json.Marshal(wrapper)
	ws.HubInstance.Broadcast <- msg
	// Response API
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
	var req models.ExtendedPasienRequest
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
	// Parse tanggal lahir
	parsedDate, err := time.Parse("2006-01-02", req.TanggalLahir)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid date format for tanggal_lahir. Use YYYY-MM-DD",
			"data":    nil,
		})
	}

	// Cari id_agama berdasarkan nama agama
	var idAgama int
	err = pc.Service.DB.QueryRow("SELECT id_agama FROM Agama WHERE nama = ?", req.Agama).Scan(&idAgama)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  http.StatusBadRequest,
				"message": "Agama not found",
				"data":    nil,
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to find agama: " + err.Error(),
			"data":    nil,
		})
	}

	// Konversi status_perkawinan
	var statusPerkawinan int
	switch strings.ToLower(req.StatusPerkawinan) {
	case "sudah kawin":
		statusPerkawinan = 1
	case "belum kawin":
		statusPerkawinan = 0
	default:
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid status_perkawinan. Must be 'sudah kawin' or 'belum kawin'",
			"data":    nil,
		})
	}

	// Siapkan model Pasien
	p := models.Pasien{
		Nama:             req.Nama,
		TanggalLahir:     parsedDate,
		JenisKelamin:     req.JenisKelamin,
		TempatLahir:      req.TempatLahir,
		NIK:              req.Nik,
		Kelurahan:        req.Kelurahan,
		Kecamatan:        req.Kecamatan,
		KotaTinggal:      req.KotaTempatTinggal,
		Alamat:           req.Alamat,
		NoTelp:           req.NoTelp,
		IDAgama:          idAgama,
		StatusPerkawinan: statusPerkawinan,
		Pekerjaan:        req.Pekerjaan,
	}

	// Panggil service dengan penanggung_jawab
	idPasien, idAntrian, nomorAntrian, idRM, idStatus, namaPoli, err :=
		pc.Service.UpdatePasienAndRegisterKunjungan(p, req.IDPoli, req.KeluhanUtama, req.PenanggungJawab)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to register kunjungan: " + err.Error(),
			"data":    nil,
		})
	}

	// Broadcast WebSocket
	inner := map[string]interface{}{
		"id_antrian":     idAntrian,
		"id_pasien":      idPasien,
		"id_poli":        req.IDPoli,
		"id_rm":          idRM,
		"id_status":      idStatus,
		"nama":           req.Nama,
		"nama_poli":      namaPoli,
		"nomor_antrian":  nomorAntrian,
		"priority_order": nomorAntrian,
		"status":         "Menunggu",
	}
	wrapper := map[string]interface{}{
		"type": "antrian_update",
		"data": inner,
	}
	msg, _ := json.Marshal(wrapper)
	ws.HubInstance.Broadcast <- msg

	// Response
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

    if err := pc.Service.TundaPasien(idAntrian); err != nil {
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

    // Siapkan payload broadcast dengan wrapper "type" & "data"
    inner := map[string]interface{}{
        "id_antrian": idAntrian,
        "status":     "Ditunda",
    }
    wrapper := map[string]interface{}{
        "type": "antrian_update",
        "data": inner,
    }

    messageJSON, err := json.Marshal(wrapper)
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

    // Siapkan payload broadcast dengan wrapper "type" & "data"
    inner := map[string]interface{}{
        "id_antrian":         idAntrian,
        "new_priority_order": newPriority,
        "status":             "Menunggu",
    }
    wrapper := map[string]interface{}{
        "type": "antrian_update",
        "data": inner,
    }

    messageJSON, err := json.Marshal(wrapper)
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
    if err := pc.Service.BatalkanAntrian(idAntrian); err != nil {
        if strings.Contains(err.Error(), "tidak ditemukan") {
            return c.JSON(http.StatusNotFound, map[string]interface{}{
                "status":  http.StatusNotFound,
                "message": err.Error(),
                "data":    nil,
            })
        }
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Gagal membatalkan antrian: " + err.Error(),
            "data":    nil,
        })
    }

    // 4. Siapkan payload broadcast dengan wrapper "type" & "data"
    inner := map[string]interface{}{
        "id_antrian": idAntrian,
        "status":     "Dibatalkan",
    }
    wrapper := map[string]interface{}{
        "type": "antrian_update",
        "data": inner,
    }

    msg, err := json.Marshal(wrapper)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Gagal membuat pesan broadcast: " + err.Error(),
            "data":    nil,
        })
    }
    ws.HubInstance.Broadcast <- msg

    // 5. Respons sukses
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Antrian berhasil dibatalkan",
        "data":    nil,
    })
}


func (pc *PasienController) GetDetailAntrianHandler(c echo.Context) error {
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

    detail, err := pc.Service.GetDetailAntrianByID(idAntrian)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve detail antrian: " + err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Detail antrian retrieved successfully",
        "data":    detail,
    })
}

// GetAgamaList handles the GET request to fetch the list of religions
func (pc *PasienController) GetAgamaList(c echo.Context) error {
	agamaList, err := pc.Service.GetAgamaList()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve agama list: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Agama list retrieved successfully",
		"data":    agamaList,
	})
}