package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
)

type AdministrasiController struct {
	Service *services.AdministrasiService
}

func NewAdministrasiController(service *services.AdministrasiService) *AdministrasiController {
	return &AdministrasiController{Service: service}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (ac *AdministrasiController) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload",
			"data":    nil,
		})
	}

	admin, err := ac.Service.AuthenticateAdmin(req.Username, req.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid username or password",
			"data":    nil,
		})
	}

	// Set exp token untuk administrasi dengan durasi yang lama
	expTime := time.Now().Add(999999 * time.Hour)
	token, err := utils.GenerateJWTToken(
		strconv.Itoa(admin.ID_Admin),
		"Administrasi",
		admin.ID_Role,
		admin.Privileges,
		0, // idPoli tidak berlaku untuk administrasi
		admin.Username,
		expTime,
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to generate token: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Login successful",
		"data": map[string]interface{}{
			"id":       admin.ID_Admin,
			"nama":     admin.Nama,
			"username": admin.Username,
			"token":    token,
		},
	})
}