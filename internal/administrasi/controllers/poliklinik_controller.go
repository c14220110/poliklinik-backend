package controllers

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	"github.com/labstack/echo/v4"
)

type PoliklinikController struct {
    Service *services.PoliklinikService
}

func NewPoliklinikController(service *services.PoliklinikService) *PoliklinikController {
    return &PoliklinikController{Service: service}
}

func (pc *PoliklinikController) GetPoliklinikList(c echo.Context) error {
    results, err := pc.Service.GetPoliklinikList()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve poliklinik list: " + err.Error(),
            "data":    nil,
        })
    }
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Poliklinik list retrieved successfully",
        "data":    results,
    })
}