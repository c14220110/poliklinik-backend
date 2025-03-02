package controllers

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/labstack/echo/v4"
)

type PrivilegeController struct {
	Service *services.PrivilegeService
}

func NewPrivilegeController(service *services.PrivilegeService) *PrivilegeController {
	return &PrivilegeController{Service: service}
}

// GetAllPrivilegesHandler mengembalikan seluruh data privilege dalam bentuk JSON
func (pc *PrivilegeController) GetAllPrivilegesHandler(c echo.Context) error {
	privileges, err := pc.Service.GetAllPrivileges()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve privileges: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Privileges retrieved successfully",
		"data":    privileges,
	})
}
