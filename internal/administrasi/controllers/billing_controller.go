package controllers

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
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