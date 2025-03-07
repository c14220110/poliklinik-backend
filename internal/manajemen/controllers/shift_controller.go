package controllers

import (
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
)

type ShiftController struct {
	Service *services.ShiftService
	// Jika tidak ingin menerima DB langsung, sebaiknya dependency db dihapus
	// Namun jika diperlukan untuk fungsi tertentu, sebaiknya dipisahkan
	// DB *sql.DB
}

func NewShiftController(service *services.ShiftService /*, db *sql.DB */) *ShiftController {
	return &ShiftController{
		Service: service,
		// DB: db,
	}
}

// AssignShiftHandler menerima query parameter id_poli, id_karyawan, id_role,
// dan request body berisi tanggal dan id_shift.
// id_management diambil dari JWT.
func (sc *ShiftController) AssignShiftHandler(c echo.Context) error {
	// Ambil query parameter
	idPoliStr := c.QueryParam("id_poli")
	idKaryawanStr := c.QueryParam("id_karyawan")
	idRoleStr := c.QueryParam("id_role")
	if idPoliStr == "" || idKaryawanStr == "" || idRoleStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli, id_karyawan, dan id_role harus disediakan",
			"data":    nil,
		})
	}

	idPoli, err := strconv.Atoi(idPoliStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli harus berupa angka",
			"data":    nil,
		})
	}
	idKaryawan, err := strconv.Atoi(idKaryawanStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_karyawan harus berupa angka",
			"data":    nil,
		})
	}
	idRole, err := strconv.Atoi(idRoleStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_role harus berupa angka",
			"data":    nil,
		})
	}

	// Ambil data dari request body: tanggal dan id_shift
	var req struct {
		Tanggal string `json:"tanggal"` // Format "YYYY-MM-DD"
		IdShift int    `json:"id_shift"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "invalid request payload: " + err.Error(),
			"data":    nil,
		})
	}
	if req.Tanggal == "" || req.IdShift == 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "tanggal dan id_shift harus disediakan",
			"data":    nil,
		})
	}

	// Ambil id_management dari JWT
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "invalid or missing token claims",
			"data":    nil,
		})
	}
	idManagement, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idManagement <= 0 {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "invalid management id in token",
			"data":    nil,
		})
	}

	// Panggil fungsi service untuk assign shift dan insert ke Management_Shift_Karyawan
	idShiftKaryawan, err := sc.Service.AssignShift(idPoli, idKaryawan, idRole, req.IdShift, idManagement, req.Tanggal)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "failed to assign shift: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "shift berhasil ditambahkan",
		"data": map[string]interface{}{
			"id_shift_karyawan": idShiftKaryawan,
		},
	})
}

// UpdateCustomShiftHandler menerima query parameter id_shift_karyawan dan request body berisi
// custom_jam_mulai dan custom_jam_selesai. Handler akan melakukan validasi agar waktu custom berada dalam rentang shift.
func (sc *ShiftController) UpdateCustomShiftHandler(c echo.Context) error {
	// Ambil id_shift_karyawan dari query parameter
	idShiftKaryawanStr := c.QueryParam("id_shift_karyawan")
	if idShiftKaryawanStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_shift_karyawan harus disediakan",
			"data":    nil,
		})
	}
	idShiftKaryawan, err := strconv.Atoi(idShiftKaryawanStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_shift_karyawan harus berupa angka",
			"data":    nil,
		})
	}

	// Ambil data dari request body
	var req struct {
		CustomJamMulai   string `json:"custom_jam_mulai"`   // Format "15:04:05"
		CustomJamSelesai string `json:"custom_jam_selesai"` // Format "15:04:05"
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "invalid request payload: " + err.Error(),
			"data":    nil,
		})
	}
	if req.CustomJamMulai == "" || req.CustomJamSelesai == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "custom_jam_mulai dan custom_jam_selesai harus disediakan",
			"data":    nil,
		})
	}

	// Panggil fungsi service untuk update custom shift
	err = sc.Service.UpdateCustomShift(idShiftKaryawan, req.CustomJamMulai, req.CustomJamSelesai)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "failed to update custom shift: " + err.Error(),
			"data":    nil,
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "custom shift berhasil diperbarui",
		"data":    nil,
	})
}

func (sc *ShiftController) SoftDeleteShiftHandler(c echo.Context) error {
	// Ambil id_shift_karyawan dari query parameter
	idShiftKaryawanStr := c.QueryParam("id_shift_karyawan")
	if idShiftKaryawanStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_shift_karyawan harus disediakan",
			"data":    nil,
		})
	}
	idShiftKaryawan, err := strconv.Atoi(idShiftKaryawanStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_shift_karyawan harus berupa angka",
			"data":    nil,
		})
	}

	// Ambil id_management dari JWT
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "invalid or missing token claims",
			"data":    nil,
		})
	}
	idManagement, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idManagement <= 0 {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "invalid management id in token",
			"data":    nil,
		})
	}

	// Panggil fungsi service untuk soft delete
	err = sc.Service.SoftDeleteShiftKaryawan(idShiftKaryawan, idManagement)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "failed to soft delete shift: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Shift karyawan berhasil di soft delete",
		"data":    nil,
	})
}

func (spc *ShiftController) GetShiftPoliList(c echo.Context) error {
	list, err := spc.Service.GetShiftPoliList()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve shift poli list: " + err.Error(),
			"data":    nil,
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Shift poli list retrieved successfully",
		"data":    list,
	})
}