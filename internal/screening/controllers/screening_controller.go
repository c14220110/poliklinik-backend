package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/screening/models"
	"github.com/c14220110/poliklinik-backend/internal/screening/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/c14220110/poliklinik-backend/ws"
	"github.com/labstack/echo/v4"
)

type ScreeningController struct {
	Service *services.ScreeningService
}

func NewScreeningController(service *services.ScreeningService) *ScreeningController {
	return &ScreeningController{Service: service}
}

// InputScreeningRequest merepresentasikan struktur data dari body request
type InputScreeningRequest struct {
	Systolic       int     `json:"systolic"`
	Diastolic      int     `json:"diastolic"`
	Berat_Badan    float64 `json:"berat_badan"`
	Suhu_Tubuh     float64 `json:"suhu_tubuh"`
	Tinggi_Badan   float64 `json:"tinggi_badan"`
	Detak_Nadi     int     `json:"detak_nadi"`
	Laju_Respirasi int     `json:"laju_respirasi"`
	Keterangan     string  `json:"keterangan"`
}

func (sc *ScreeningController) InputScreening(c echo.Context) error {
    // Ambil claims dari token (misalnya ID_Karyawan operator)
    claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
    if !ok || claims == nil {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid or missing token claims",
            "data":    nil,
        })
    }

    operatorID, err := strconv.Atoi(claims.IDKaryawan)
    if err != nil || operatorID <= 0 {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid operator ID in token",
            "data":    nil,
        })
    }

    // Ambil id_antrian dari query parameter
    idAntrianStr := c.QueryParam("id_antrian")
    if idAntrianStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_antrian query parameter is required",
            "data":    nil,
        })
    }
    idAntrian, err := strconv.Atoi(idAntrianStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "Invalid id_antrian parameter",
            "data":    nil,
        })
    }

    // Bind request body ke struct
    var req InputScreeningRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "Invalid request payload: " + err.Error(),
            "data":    nil,
        })
    }

    // Mapping request ke ScreeningInput
    screeningInput := models.ScreeningInput{
        Systolic:       req.Systolic,
        Diastolic:      req.Diastolic,
        Berat_Badan:    req.Berat_Badan,
        Suhu_Tubuh:     req.Suhu_Tubuh,
        Tinggi_Badan:   req.Tinggi_Badan,
        Detak_Nadi:     req.Detak_Nadi,
        Laju_Respirasi: req.Laju_Respirasi,
        Keterangan:     req.Keterangan,
    }

    // Panggil service untuk menyimpan data
    screeningID, err := sc.Service.InputScreening(screeningInput, idAntrian, operatorID)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to input screening: " + err.Error(),
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

    messageJSON, err := json.Marshal(wrapper)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to marshal broadcast message: " + err.Error(),
            "data":    nil,
        })
    }
    ws.HubInstance.Broadcast <- messageJSON

    // Kembalikan response sukses
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