package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/screening/models"
	"github.com/c14220110/poliklinik-backend/internal/screening/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
)

// ScreeningController menangani endpoint terkait screening.
type ScreeningController struct {
    Service *services.ScreeningService
}

// NewScreeningController membuat instance ScreeningController.
func NewScreeningController(service *services.ScreeningService) *ScreeningController {
    return &ScreeningController{Service: service}
}

// InputScreeningRequest mendefinisikan struktur data request untuk input screening.
type InputScreeningRequest struct {
    ID_Pasien      int     `json:"id_pasien"`
    Tensi          string  `json:"tensi"`
    Berat_Badan    int     `json:"berat_badan"`
    Suhu_Tubuh     float64 `json:"suhu_tubuh"`
    Tinggi_Badan   float64 `json:"tinggi_badan"`
    Gula_Darah     float64 `json:"gula_darah"`
    Detak_Nadi     int     `json:"detak_nadi"`
    Laju_Respirasi int     `json:"laju_respirasi"`
    Keterangan     string  `json:"keterangan"`
}

func (sc *ScreeningController) InputScreening(c echo.Context) error {
    // Ambil klaim JWT dari context
    claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
    if !ok || claims == nil {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid or missing operator ID in token",
            "data":    nil,
        })
    }

    // Konversi claims.IDKaryawan ke int
    operatorID, err := strconv.Atoi(claims.IDKaryawan)
    if err != nil || operatorID <= 0 {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid operator ID in token",
            "data":    nil,
        })
    }

    var req InputScreeningRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "Invalid request payload: " + err.Error(),
            "data":    nil,
        })
    }
    if req.ID_Pasien <= 0 {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "ID_Pasien must be provided and greater than 0",
            "data":    nil,
        })
    }

    // Buat objek Screening
    screening := models.Screening{
        ID_Pasien:      req.ID_Pasien,
        ID_Karyawan:    operatorID,
        Tensi:          req.Tensi,
        Berat_Badan:    req.Berat_Badan,
        Suhu_Tubuh:     req.Suhu_Tubuh,
        Tinggi_Badan:   req.Tinggi_Badan,
        Gula_Darah:     req.Gula_Darah,
        Detak_Nadi:     req.Detak_Nadi,
        Laju_Respirasi: req.Laju_Respirasi,
        Keterangan:     req.Keterangan,
        Created_At:     time.Now(),
    }

    // Panggil service untuk memasukkan data screening dan update Riwayat_Kunjungan
    screeningID, err := sc.Service.InputScreening(screening)
    if err != nil {
        if err.Error() == "failed to get Riwayat_Kunjungan: sql: no rows in result set" {
            return c.JSON(http.StatusConflict, map[string]interface{}{
                "status":  http.StatusConflict,
                "message": "Screening has already been recorded for this visit",
                "data":    nil,
            })
        }
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to input screening: " + err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Screening recorded successfully",
        "data": map[string]interface{}{
            "ID_Screening": screeningID,
        },
    })
}

// GetScreeningByPasienHandler mengembalikan seluruh record screening untuk pasien berdasarkan query parameter id_pasien.
func (sc *ScreeningController) GetScreeningByPasienHandler(c echo.Context) error {
    idPasienParam := c.QueryParam("id_pasien")
    if idPasienParam == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_pasien parameter is required",
            "data":    nil,
        })
    }

    idPasien, err := strconv.Atoi(idPasienParam)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_pasien must be a number",
            "data":    nil,
        })
    }

    screenings, err := sc.Service.GetScreeningByPasien(idPasien)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve screening records: " + err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Screening records retrieved successfully",
        "data":    screenings,
    })
}