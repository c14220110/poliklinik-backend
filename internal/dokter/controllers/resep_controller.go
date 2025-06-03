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
	idKaryawan := claims.IDKaryawan

	// --- validasi minimal ---
	if req.IDKunjungan == 0 || len(req.Sections) == 0 {
			return c.JSON(http.StatusBadRequest, echo.Map{
					"status":  http.StatusBadRequest,
					"message": "id_kunjungan and sections are required",
					"data":    nil,
			})
	}

	// --- panggil service ---
	result, err := rc.Service.CreateResep(req, idKaryawan)
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
			"data":    result,
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

func (rc *ResepController) GetRiwayatKunjunganHandler(c echo.Context) error {
	idPasienParam := c.QueryParam("id_pasien")
	if idPasienParam == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"status":  http.StatusBadRequest,
					"message": "id_pasien parameter is required",
					"data":    nil,
			})
	}

	idPasien, err := strconv.Atoi(idPasienParam)
	if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"status":  http.StatusBadRequest,
					"message": "id_pasien must be a number",
					"data":    nil,
			})
	}

	riwayat, err := rc.Service.GetRiwayatKunjunganByPasien(idPasien)
	if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"status":  http.StatusInternalServerError,
					"message": "Failed to retrieve riwayat kunjungan: " + err.Error(),
					"data":    nil,
			})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  http.StatusOK,
			"message": "Riwayat kunjungan retrieved successfully",
			"data":    riwayat,
	})
}

func (rc *ResepController) GetICD9CMList(c echo.Context) error {
	q        := c.QueryParam("q")                    // search display LIKE
	limit, _ := strconv.Atoi(c.QueryParam("limit")) // default di-handle service
	page, _  := strconv.Atoi(c.QueryParam("page"))  // halaman mulai 1

	list, total, limit, err := rc.Service.GetICD9CMList(q, limit, page)
	if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"status":  http.StatusInternalServerError,
					"message": "Failed to retrieve ICD9_CM list: " + err.Error(),
					"data":    nil,
			})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  http.StatusOK,
			"message": "ICD9_CM list retrieved successfully",
			"data": map[string]interface{}{
					"list":  list,
					"total": total,
					"limit": limit,
			},
	})
}

func (rc *ResepController) GetICD9List(c echo.Context) error {
	q        := c.QueryParam("q")                    // search display LIKE
	limit, _ := strconv.Atoi(c.QueryParam("limit")) // default di-handle service
	page, _  := strconv.Atoi(c.QueryParam("page"))  // halaman mulai 1

	list, total, limit, err := rc.Service.GetICD9CMList(q, limit, page)
	if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"status":  http.StatusInternalServerError,
					"message": "Failed to retrieve ICD9 list: " + err.Error(),
					"data":    nil,
			})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  http.StatusOK,
			"message": "ICD9 list retrieved successfully",
			"data": map[string]interface{}{
					"list":  list,
					"total": total,
					"limit": limit,
			},
	})
}