package controllers

import (
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	"github.com/labstack/echo/v4"
)

// BillingController menangani permintaan terkait data Billing.
type BillingController struct {
    Service *services.BillingService
}

func NewBillingController(service *services.BillingService) *BillingController {
    return &BillingController{Service: service}
}

// ListBilling mengembalikan data billing dengan struktur:
// { "status": HTTP_CODE, "message": "Feedback", "data": [ ... ] }
func (bc *BillingController) ListBilling(c echo.Context) error {
    statusParam := c.QueryParam("status")
    var filterStatus *int
    if statusParam != "" {
        if val, err := strconv.Atoi(statusParam); err == nil {
            filterStatus = &val
        }
    }

    data, err := bc.Service.GetRecentBilling(filterStatus)
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
        "data":    data,
    })
}

// BillingDetail mengembalikan detail billing berdasarkan id_pasien
func (bc *BillingController) BillingDetail(c echo.Context) error {
    idParam := c.QueryParam("id_pasien")
    if idParam == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_pasien is required",
            "data":    nil,
        })
    }
    id, err := strconv.Atoi(idParam)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "Invalid id_pasien",
            "data":    nil,
        })
    }

    detail, err := bc.Service.GetBillingDetail(id)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve billing detail: " + err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, detail)
}