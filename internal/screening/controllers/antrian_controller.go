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
        if strings.Contains(err.Error(), "tidak ada pasien") {
            return c.JSON(http.StatusNotFound, map[string]interface{}{
                "status":  http.StatusNotFound,
                "message": err.Error(),
                "data":    nil,
            })
        }
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to update antrian: " + err.Error(),
            "data":    nil,
        })
    }

    // Siapkan payload broadcast dengan wrapper "type" & "data"
    inner := map[string]interface{}{
        "id_antrian": result["id_antrian"],
        "status":     "Screening",
    }
    wrapper := map[string]interface{}{
        "type": "antrian_update",
        "data": inner,
    }

    msg, err := json.Marshal(wrapper)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to marshal broadcast message: " + err.Error(),
            "data":    nil,
        })
    }
    ws.HubInstance.Broadcast <- msg

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
        if strings.Contains(err.Error(), "tidak ada pasien") {
            return c.JSON(http.StatusNotFound, map[string]interface{}{
                "status":  http.StatusNotFound,
                "message": err.Error(),
                "data":    nil,
            })
        }
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to update antrian: " + err.Error(),
            "data":    nil,
        })
    }

    // Siapkan payload broadcast dengan wrapper "type" & "data"
    inner := map[string]interface{}{
        "id_antrian": result["id_antrian"],
        "status":     "Konsultasi",
    }
    wrapper := map[string]interface{}{
        "type": "antrian_update",
        "data": inner,
    }

    msg, err := json.Marshal(wrapper)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to marshal broadcast message: " + err.Error(),
            "data":    nil,
        })
    }
    ws.HubInstance.Broadcast <- msg

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
    if err := ac.AntrianService.PulangkanPasien(idAntrian); err != nil {
        if strings.Contains(err.Error(), "tidak ditemukan") {
            return c.JSON(http.StatusNotFound, map[string]interface{}{
                "status":  http.StatusNotFound,
                "message": err.Error(),
                "data":    nil,
            })
        }
        if strings.Contains(err.Error(), "status antrian saat ini bukan Konsultasi") {
            return c.JSON(http.StatusBadRequest, map[string]interface{}{
                "status":  http.StatusBadRequest,
                "message": err.Error(),
                "data":    nil,
            })
        }
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to pulangkan pasien: " + err.Error(),
            "data":    nil,
        })
    }

    // Siapkan payload broadcast dengan wrapper "type" & "data"
    inner := map[string]interface{}{
        "id_antrian": idAntrian,
        "status":     "Selesai",
    }
    wrapper := map[string]interface{}{
        "type": "antrian_update",
        "data": inner,
    }

    msg, err := json.Marshal(wrapper)
    if err != nil {
        // log error internal saja, tidak menggagalkan respons
        // log.Println("Failed to marshal broadcast message:", err)
    } else {
        ws.HubInstance.Broadcast <- msg
    }

    // Kembalikan respons sukses
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Pasien berhasil dipulangkan",
        "data": map[string]interface{}{
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
	if err := ac.AntrianService.AlihkanPasien(idAntrian); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Gagal mengalihkan pasien: " + err.Error(),
			"data":    nil,
		})
	}

	// Siapkan payload broadcast dengan wrapper "type" & "data"
	inner := map[string]interface{}{
		"id_antrian": idAntrian,
		"status":     "Pra-Konsultasi",
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


func (ac *AntrianController) GetDetailAntrianHandler(c echo.Context) error {
	// Ambil query parameter id_antrian
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

	// Panggil service untuk mendapatkan detail antrian berdasarkan id_antrian.
	result, err := ac.AntrianService.GetDetailAntrianByID(idAntrian)
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
		"data":    result,
	})
}
