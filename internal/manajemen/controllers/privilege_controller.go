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

// RequestCreatePrivilege adalah struct untuk request body pada POST privilege
type RequestCreatePrivilege struct {
	NamaPrivilege string `json:"nama_privilege"`
	Deskripsi     string `json:"deskripsi"`
}

// CreatePrivilegeHandler menangani request POST untuk membuat privilege baru
func (pc *PrivilegeController) CreatePrivilegeHandler(c echo.Context) error {
	var req RequestCreatePrivilege
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Failed to bind request: " + err.Error(),
			"data":    nil,
		})
	}

	err := pc.Service.CreatePrivilege(req.NamaPrivilege, req.Deskripsi)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to create privilege: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"status":  http.StatusCreated,
		"message": "Privilege created successfully",
		"data":    nil,
	})
}