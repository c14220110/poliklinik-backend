package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/c14220110/poliklinik-backend/internal/screening/services"
	"github.com/c14220110/poliklinik-backend/ws"
	"github.com/labstack/echo/v4"
)

type AntrianController struct {
	AntrianService *services.AntrianService
}

func NewAntrianController(service *services.AntrianService) *AntrianController {
	return &AntrianController{AntrianService: service}
}

func (ac *AntrianController) MasukkanPasienHandler(c echo.Context) error {
	// Ambil parameter id_poli dari query string.
	idPoliStr := c.QueryParam("id_poli")
	if idPoliStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli parameter is required",
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

	// Panggil service untuk mengubah status antrian dan mendapatkan detail data pasien.
	result, err := ac.AntrianService.MasukkanPasien(idPoli)
	if err != nil {
		// Jika error mengindikasikan tidak ada data yang ditemukan, kembalikan 404.
		if strings.Contains(err.Error(), "tidak ada pasien") {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"status":  http.StatusNotFound,
				"message": err.Error(),
				"data":    nil,
			})
		}
		// Untuk error lain, kembalikan status 500.
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to update antrian: " + err.Error(),
			"data":    nil,
		})
	}

	// Tambahkan broadcast ke WebSocket dengan format { id_antrian: ..., status: "Screening" }
	broadcastData := map[string]interface{}{
		"id_antrian": result["id_antrian"],
		"status":     "Screening",
	}
	messageJSON, err := json.Marshal(broadcastData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to marshal broadcast message: " + err.Error(),
			"data":    nil,
		})
	}
	ws.HubInstance.Broadcast <- messageJSON

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Pasien berhasil dimasukkan",
		"data":    result,
	})
}



// GetAntrianTerlamaHandler menangani request untuk mendapatkan antrian pasien paling lama dengan status = 0
func (ac *AntrianController) GetAntrianTerlamaHandler(c echo.Context) error {
    idPoliParam := c.QueryParam("id_poli")
    if idPoliParam == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_poli harus diberikan",
            "data":    nil,
        })
    }

    idPoli, err := strconv.Atoi(idPoliParam)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_poli harus berupa angka",
            "data":    nil,
        })
    }

    // Perbaiki: gunakan ac.AntrianService, bukan ac.Service
    data, err := ac.AntrianService.GetAntrianTerlama(idPoli)
    if err != nil {
        return c.JSON(http.StatusNotFound, map[string]interface{}{
            "status":  http.StatusNotFound,
            "message": err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Antrian ditemukan",
        "data":    data,
    })
}

func (ac *AntrianController) MasukkanPasienKeDokterHandler(c echo.Context) error {
	// Ambil parameter id_poli dari query string.
	idPoliStr := c.QueryParam("id_poli")
	if idPoliStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli parameter is required",
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

	// Panggil service untuk mengubah status antrian dan mendapatkan detail data pasien.
	result, err := ac.AntrianService.MasukkanPasienKeDokter(idPoli)
	if err != nil {
		// Jika error mengindikasikan tidak ada data yang ditemukan, kembalikan 404.
		if strings.Contains(err.Error(), "tidak ada pasien") {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"status":  http.StatusNotFound,
				"message": err.Error(),
				"data":    nil,
			})
		}
		// Untuk error lain, kembalikan status 500.
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to update antrian: " + err.Error(),
			"data":    nil,
		})
	}

	// Broadcast ke WebSocket: { id_antrian: <id_antrian>, status: "Konsultasi" }
	broadcastData := map[string]interface{}{
		"id_antrian": result["id_antrian"],
		"status":     "Konsultasi",
	}
	messageJSON, err := json.Marshal(broadcastData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to marshal broadcast message: " + err.Error(),
			"data":    nil,
		})
	}
	ws.HubInstance.Broadcast <- messageJSON

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Pasien berhasil dimasukkan",
		"data":    result,
	})
}

// PulangkanPasienHandler menangani request untuk memulangkan pasien
// dengan mengubah status antrian dari 5 menjadi 6.
func (ac *AntrianController) PulangkanPasienHandler(c echo.Context) error {
	// Ambil parameter id_antrian dari query string
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

	// Panggil service untuk memulangkan pasien
	err = ac.AntrianService.PulangkanPasien(idAntrian)
	if err != nil {
		// Jika antrian tidak ditemukan
		if strings.Contains(err.Error(), "tidak ditemukan") {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"status":  http.StatusNotFound,
				"message": err.Error(),
				"data":    nil,
			})
		}
		// Jika status bukan 5
		if strings.Contains(err.Error(), "status antrian saat ini bukan Konsultasi") {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  http.StatusBadRequest,
				"message": err.Error(),
				"data":    nil,
			})
		}
		// Error lainnya
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to pulangkan pasien: " + err.Error(),
			"data":    nil,
		})
	}

	// Broadcast ke WebSocket: { id_antrian: <id_antrian>, status: "Pulang" }
	broadcastData := map[string]interface{}{
		"id_antrian": idAntrian,
		"status":     "Selesai",
	}
	messageJSON, err := json.Marshal(broadcastData)
	if err != nil {
		// Log error secara internal, tidak kembalikan ke client
		// Misalnya: log.Println("Failed to marshal broadcast message:", err)
	} else {
		ws.HubInstance.Broadcast <- messageJSON
	}

	// Kembalikan respons sukses
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Pasien berhasil dipulangkan",
		"data":    map[string]interface{}{
			"id_antrian": idAntrian,
		},
	})
}

func (ac *AntrianController) AlihkanPasienHandler(c echo.Context) error {
	// Ambil parameter id_antrian dari query string.
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

	// Panggil service untuk mengubah status antrian menjadi 4.
	err = ac.AntrianService.AlihkanPasien(idAntrian)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Gagal mengalihkan pasien: " + err.Error(),
			"data":    nil,
		})
	}

	// Broadcast ke WebSocket dengan format { id_antrian: <id_antrian>, status: "Pra-Konsultasi" }
	broadcastData := map[string]interface{}{
		"id_antrian": idAntrian,
		"status":     "Pra-Konsultasi",
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

	// Kembalikan response sukses
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Pasien berhasil dialihkan",
		"data":    nil,
	})
}
func (ac *AntrianController) GetTodayScreeningAntrianHandler(c echo.Context) error {
	// Ambil query parameter id_poli dari query string.
	idPoliStr := c.QueryParam("id_poli")
	if idPoliStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli query parameter is required",
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

	// Panggil service untuk mengambil data screening antrian untuk tanggal sekarang.
	results, err := ac.AntrianService.GetTodayScreeningAntrianByPoli(idPoli)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve screening antrian: " + err.Error(),
			"data":    nil,
		})
	}

	// Kembalikan response dengan data berupa array of object
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Screening antrian retrieved successfully",
		"data":    results,
	})
}
