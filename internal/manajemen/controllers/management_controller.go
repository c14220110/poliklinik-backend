package controllers

import (
	"net/http"
	"strconv"

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

    // Autentikasi manajemen melalui service
    m, err := mc.Service.AuthenticateManagement(req.Username, req.Password)
    if err != nil {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid username or password",
            "data":    nil,
        })
    }

    // Siapkan privileges untuk management
    extraClaims := []map[string]interface{}{
        {"privilege": "manage_dashboard"},
        {"privilege": "manage_poli"},
        {"privilege": "manage_shift"},
        {"privilege": "manage_user"},
        {"privilege": "manage_role_privilege"},
    }

    // Gunakan GenerateJWTToken dari utils
    token, err := utils.GenerateJWTToken(
        strconv.Itoa(m.ID_Management),
        "Manajemen",
        extraClaims,
        0,
        m.Username,
    )
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to generate token: " + err.Error(),
            "data":    nil,
        })
    }

    // Kirim response standar
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