package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/c14220110/poliklinik-backend/internal/screening/services"
	"github.com/labstack/echo/v4"
)

type AntrianController struct {
	AntrianService *services.AntrianService
}

func NewAntrianController(service *services.AntrianService) *AntrianController {
	return &AntrianController{AntrianService: service}
}

// MasukkanPasienHandler menangani request untuk mengubah status antrian pasien.
// Jika tidak ada baris dengan id_status = 0 atau id_poli tidak valid, handler akan mengembalikan response 404.
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

	// Panggil service yang sudah diperbarui untuk mengubah status antrian
	// dan mengembalikan detail data pasien.
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

	// Panggil service yang sudah diperbarui untuk mengubah status antrian
	// dan mengembalikan detail data pasien.
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

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Pasien berhasil dimasukkan",
		"data":    result,
	})
}