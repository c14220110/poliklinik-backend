package controllers

import (
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
)

type BillingController struct {
	Service *services.BillingService
}

func NewBillingController(service *services.BillingService) *BillingController {
	return &BillingController{Service: service}
}

func (bc *BillingController) ListBilling(c echo.Context) error {
	// Ambil query parameter id_poli dan status (jika ada)
	idPoliFilter := c.QueryParam("id_poli")
	statusFilter := c.QueryParam("status")

	list, err := bc.Service.GetBillingData(idPoliFilter, statusFilter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve billing data: " + err.Error(),
			"data":    nil,
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Billing data retrieved successfully",
		"data":    list,
	})
}

func (bc *BillingController) InputBillingAssessment(c echo.Context) error {
	// -------- query param ----------
	idAntrianStr    := c.QueryParam("id_antrian")
	idAssessmentStr := c.QueryParam("id_assessment")
	if idAntrianStr == "" || idAssessmentStr == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "id_antrian & id_assessment are required",
			"data":    nil,
		})
	}
	idAntrian, err := strconv.Atoi(idAntrianStr)
	idAssessment, err2 := strconv.Atoi(idAssessmentStr)
	if err != nil || err2 != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "invalid query‑param (must be integer)",
			"data":    nil,
		})
	}

	// -------- JWT claims ----------
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
	}
	idKaryawanJWT, _ := strconv.Atoi(claims.IDKaryawan)

	// -------- body ----------
	var req models.InputBillingRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
	}

	// -------- service call ----------
	if err := bc.Service.SaveBillingAssessment(
		idAntrian, idAssessment, idKaryawanJWT, req); err != nil {

		return c.JSON(http.StatusInternalServerError, echo.Map{
			"status":  http.StatusInternalServerError,
			"message": "Failed to save billing: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  http.StatusOK,
		"message": "Billing saved",
		"data":    nil,
	})
}