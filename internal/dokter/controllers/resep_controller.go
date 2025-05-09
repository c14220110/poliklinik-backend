package controllers

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/dokter/models"
	"github.com/c14220110/poliklinik-backend/internal/dokter/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
)

type ResepController struct{ Service *services.ResepService }

func NewResepController(s *services.ResepService) *ResepController {
	return &ResepController{Service: s}
}

// POST /api/dokter/resep
func (rc *ResepController) CreateResepHandler(c echo.Context) error {
	var req models.ResepRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
	}

	// --- ambil id_karyawan (dokter) dari JWT ---
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
	}
	idKaryawan, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idKaryawan <= 0 {
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"status":  http.StatusUnauthorized,
			"message": "Invalid karyawan ID in token",
			"data":    nil,
		})
	}
	req.IDKaryawan = idKaryawan // override nilai dari body (jika ada)

	// --- validasi minimal ---
	if req.IDKunjungan == 0 || len(req.Sections) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "id_kunjungan and sections are required",
			"data":    nil,
		})
	}

	// --- panggil service ---
	idResep, err := rc.Service.CreateResep(req)
	if err != nil {
		switch err {
		case services.ErrKunjunganNotFound:
			return c.JSON(http.StatusNotFound, echo.Map{
				"status":  http.StatusNotFound,
				"message": "Kunjungan tidak ditemukan",
				"data":    nil,
			})
		default:
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"status":  http.StatusInternalServerError,
				"message": "Failed to create resep: " + err.Error(),
				"data":    nil,
			})
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  http.StatusOK,
		"message": "Resep created successfully",
		"data":    echo.Map{"id_resep": idResep},
	})
}

// GET /obat?q=amox&limit=20&page=2
func (rc *ResepController) GetObatList(c echo.Context) error {
	q        := c.QueryParam("q")                    // search nama LIKE
	limit, _ := strconv.Atoi(c.QueryParam("limit")) // default di‑handle service
	page, _  := strconv.Atoi(c.QueryParam("page"))  // halaman mulai 1

	list, err := rc.Service.GetObatList(q, limit, page)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve obat list: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Obat list retrieved successfully",
		"data":    list,
	})
}