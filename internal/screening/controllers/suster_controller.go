package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/screening/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
)

type LoginSusterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IDPoli   int    `json:"id_poli"`
}

type SusterController struct {
	Service *services.SusterService
}

func NewSusterController(service *services.SusterService) *SusterController {
	return &SusterController{Service: service}
}

func (sc *SusterController) LoginSuster(c echo.Context) error {
	var req LoginSusterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
	}
	if req.Username == "" || req.Password == "" || req.IDPoli == 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Username, Password, and IDPoli are required",
			"data":    nil,
		})
	}

	suster, shift, err := sc.Service.AuthenticateSusterUsingKaryawan(req.Username, req.Password, req.IDPoli)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid username or password: " + err.Error(),
			"data":    nil,
		})
	}

	// Hitung waktu expiration berdasarkan custom_jam_selesai shift aktif.
	today := time.Now().Format("2006-01-02")
	expStr := today + " " + shift.CustomJamSelesai
	expTime, err := time.Parse("2006-01-02 15:04:05", expStr)
	if err != nil {
		// Jika gagal parsing, fallback ke 1 jam dari sekarang.
		expTime = time.Now().Add(1 * time.Hour)
	}

	// Generate token JWT dengan memasukkan nama suster ke dalam payload.
	token, err := utils.GenerateJWTToken(
		strconv.Itoa(suster.ID_Suster),
		"Suster",
		suster.ID_Role,
		suster.Privileges,
		req.IDPoli,
		suster.Username,
		suster.Nama, // Menyertakan nama suster
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
			"id":       suster.ID_Suster,
			"nama":     suster.Nama,
			"username": suster.Username,
			"role":     "Suster",
			"id_poli":  req.IDPoli,
			"token":    token,
			"shift": map[string]interface{}{
				"id_shift_karyawan": shift.ID_Shift_Karyawan,
				"jam_mulai":         shift.CustomJamMulai,
				"jam_selesai":       shift.CustomJamSelesai,
			},
		},
	})
}
