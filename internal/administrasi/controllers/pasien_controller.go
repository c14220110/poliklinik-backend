package controllers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	"github.com/gorilla/websocket"
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

var upgrader = websocket.Upgrader{
	// Izinkan origin apa saja; sesuaikan untuk production
	CheckOrigin: func(r *http.Request) bool { return true },
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

    // Panggil service dengan tiga nilai kembalian
    patientID, idAntrian, nomorAntrian, err := pc.Service.RegisterPasienWithKunjungan(p, req.IDPoli, operatorID, req.KeluhanUtama)
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

    // Kembalikan respons dengan id_pasien, id_antrian, dan nomor_antrian
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

    idPasien, idAntrian, nomorAntrian, err := pc.Service.UpdatePasienAndRegisterKunjungan(p, req.IDPoli, req.KeluhanUtama)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to register kunjungan: " + err.Error(),
            "data":    nil,
        })
    }

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
            "message": "id_antrian parameter is required",
            "data":    nil,
        })
    }
    idAntrian, err := strconv.Atoi(idAntrianStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_antrian must be a number",
            "data":    nil,
        })
    }

    err = pc.Service.TundaPasien(idAntrian)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to tunda pasien: " + err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Pasien successfully tunda",
        "data":    nil,
    })
}

func (pc *PasienController) RescheduleAntrianHandler(c echo.Context) error {
    idAntrianStr := c.QueryParam("id_antrian")
    idPoliStr := c.QueryParam("id_poli")
    if idAntrianStr == "" || idPoliStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_antrian and id_poli parameters are required",
            "data":    nil,
        })
    }
    idAntrian, err := strconv.Atoi(idAntrianStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_antrian must be a number",
            "data":    nil,
        })
    }
    idPoli, err := strconv.Atoi(idPoliStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_poli must be a number",
            "data":    nil,
        })
    }

    newPriority, err := pc.Service.RescheduleAntrianPriority(idAntrian, idPoli)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to reschedule antrian: " + err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Antrian rescheduled successfully",
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

func (pc *PasienController) GetAntrianTodayWebSocket(c echo.Context) error {
	// Upgrade koneksi HTTP ke WebSocket
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	// Buat ticker untuk mengirim data tiap 10 detik
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Ambil parameter status jika ada
			statusFilter := c.QueryParam("status")
			// Panggil service untuk mendapatkan data antrian hari ini
			list, err := pc.Service.GetAntrianToday(statusFilter)
			if err != nil {
				ws.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
				continue
			}

			// Serialisasi data ke JSON
			message, err := json.Marshal(list)
			if err != nil {
				ws.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
				continue
			}

			// Kirim data melalui WebSocket
			if err := ws.WriteMessage(websocket.TextMessage, message); err != nil {
				return err
			}

			// Cetak notifikasi di terminal
			log.Printf("API GetAntrianToday berhasil dipanggil pada %s", time.Now().Format("15:04:05"))
		}
	}
}
