package controllers

import (
	"errors"
	"net/http"
	"strconv"

	//"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
)

type CMSController struct {
	Service *services.CMSService
}

func NewCMSController(service *services.CMSService) *CMSController {
	return &CMSController{Service: service}
}

func (cc *CMSController) CreateCMSHandler(c echo.Context) error {
	// 1. Bind JSON body ke model request
	var req models.CreateCMSRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
		})
	}
	if req.IDPoli == 0 || req.Title == "" || len(req.Sections) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "id_poli, title, dan sections wajib diisi",
		})
	}

	// 2. Ambil klaim JWT dari context (middleware JWT sudah menyimpan di ContextKeyClaims)
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
		})
	}
	userID, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || userID <= 0 {
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"status":  http.StatusUnauthorized,
			"message": "Invalid user ID in token",
		})
	}

	// 3. Panggil service
	mgmt := models.ManagementCMS{
		IDManagement: userID,
		CreatedBy:    userID,
		UpdatedBy:    userID,
	}
	idCMS, err := cc.Service.CreateCMSWithSections(req, mgmt)
	if err != nil {
		// Tangani duplicate-domain error dari service
		if errors.Is(err, services.ErrCMSAlreadyExists) {
			return c.JSON(http.StatusConflict, echo.Map{
				"status":  http.StatusConflict,
				"message": "CMS already exists for this poliklinik",
			})
		}
		// Error lain → 500
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"status":  http.StatusInternalServerError,
			"message": "Failed to create CMS: " + err.Error(),
		})
	}

	// 4. Response sukses
	return c.JSON(http.StatusOK, echo.Map{
		"status":  http.StatusOK,
		"message": "CMS created successfully",
		"data":    echo.Map{"id_cms": idCMS},
	})
}

func (cc *CMSController) GetCMSByPoliklinikHandler(c echo.Context) error {
    poliIDStr := c.QueryParam("id_poli")
    if poliIDStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_poli parameter is required",
            "data":    nil,
        })
    }
    poliID, err := strconv.Atoi(poliIDStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_poli must be a number",
            "data":    nil,
        })
    }

    cmsList, err := cc.Service.GetCMSByPoliklinikID(poliID)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve CMS data: " + err.Error(),
            "data":    nil,
        })
    }

    response := map[string]interface{}{
        "id_poli": poliID,
        "cms":     cmsList,
    }
    return c.JSON(http.StatusOK, response)
}

func (cc *CMSController) GetAllCMSHandler(c echo.Context) error {
    // Ambil data CMS dari service
    cmsFlatList, err := cc.Service.GetAllCMS()
    if err != nil {
        // Jika ada error, kembalikan response dengan struktur yang sama
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "data":    nil,
            "message": "Gagal mengambil data CMS: " + err.Error(),
            "status":  http.StatusInternalServerError,
        })
    }

    // Jika sukses, kembalikan response dengan data
    response := map[string]interface{}{
        "data":    cmsFlatList,
        "message": "CMS retrieved successfully",
        "status":  http.StatusOK,
    }
    return c.JSON(http.StatusOK, response)
}
