package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
)

type ManagementController struct {
	Service *services.ManagementService
}

func NewManagementController(service *services.ManagementService) *ManagementController {
	return &ManagementController{Service: service}
}

type LoginManagementRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (mc *ManagementController) Login(c echo.Context) error {
	var req LoginManagementRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload",
			"data":    nil,
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Username and Password are required",
			"data":    nil,
		})
	}

	// Autentikasi management melalui service
	m, err := mc.Service.AuthenticateManagement(req.Username, req.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid username or password",
			"data":    nil,
		})
	}

	// Karena management tidak memiliki role/privilege, kita set idRole = 0 dan privileges sebagai slice kosong.
	// Gunakan exp yang lama (misalnya, 999999 jam ke depan).
	token, err := utils.GenerateJWTToken(
		strconv.Itoa(m.ID_Management),
		"Manajemen",
		0,         // idRole tidak berlaku
		[]int{},   // privileges kosong
		0,         // idPoli tidak berlaku untuk management
		m.Username,
		time.Now().Add(999999 * time.Hour),
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
			"id":       m.ID_Management,
			"nama":     m.Nama,
			"username": m.Username,
			"token":    token,
		},
	})
}