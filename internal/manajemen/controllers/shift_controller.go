package controllers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

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
	idManagement := claims.IDKaryawan
	if idManagement <= 0 {
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
	// Ambil query parameter id_poli (opsional)
	idPoli := c.QueryParam("id_poli")
	list, err := spc.Service.GetShiftPoliList(idPoli)
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

func (sc *ShiftController) GetKaryawanListHandler(c echo.Context) error {
	// Ambil query parameter: id_poli, id_shift, id_role (opsional), dan tanggal (opsional, format DD/MM/YYYY)
	idPoli := c.QueryParam("id_poli")
	idShift := c.QueryParam("id_shift")
	idRole := c.QueryParam("id_role")
	tanggal := c.QueryParam("tanggal") // Jika kosong, otomatis hari ini

	// Validasi parameter wajib
	if idPoli == "" {
		slog.Warn("Missing id_poli parameter")
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "parameter id_poli wajib diisi",
			"data":    nil,
		})
	}
	if idShift == "" {
		slog.Warn("Missing id_shift parameter")
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "parameter id_shift wajib diisi",
			"data":    nil,
		})
	}

	list, err := sc.Service.GetListKaryawanFiltered(idPoli, idShift, idRole, tanggal)
	if err != nil {
		slog.Error("Failed to retrieve karyawan list", "error", err)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve karyawan list: " + err.Error(),
			"data":    nil,
		})
	}

	slog.Info("Successfully retrieved karyawan list", "count", len(list))
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Karyawan list retrieved successfully",
		"data":    list,
	})
}


func (mc *ShiftController) GetKaryawanTanpaShiftHandler(c echo.Context) error {
    // 1. Ambil query parameters
    idShiftStr := c.QueryParam("id_shift")
    idRoleStr := c.QueryParam("id_role")
    tanggal := c.QueryParam("tanggal")
    idPoliStr := c.QueryParam("id_poli")

    // 2. Validasi id_shift (wajib)
    if idShiftStr == "" {
        slog.Warn("Missing id_shift parameter")
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "parameter id_shift wajib diisi",
            "data":    nil,
        })
    }
    idShift, err := strconv.Atoi(idShiftStr)
    if err != nil {
        slog.Warn("Invalid id_shift format", "id_shift", idShiftStr, "error", err)
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_shift harus berupa angka",
            "data":    nil,
        })
    }

    // 3. Validasi id_poli (wajib)
    if idPoliStr == "" {
        slog.Warn("Missing id_poli parameter")
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "parameter id_poli wajib diisi",
            "data":    nil,
        })
    }
    idPoli, err := strconv.Atoi(idPoliStr)
    if err != nil {
        slog.Warn("Invalid id_poli format", "id_poli", idPoliStr, "error", err)
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_poli harus berupa angka",
            "data":    nil,
        })
    }

    // 4. Tangani id_role (opsional)
    var idRole *int
    if idRoleStr != "" {
        role, err := strconv.Atoi(idRoleStr)
        if err != nil {
            slog.Warn("Invalid id_role format", "id_role", idRoleStr, "error", err)
            return c.JSON(http.StatusBadRequest, map[string]interface{}{
                "status":  http.StatusBadRequest,
                "message": "id_role harus berupa angka",
                "data":    nil,
            })
        }
        idRole = &role
    }

    // 5. Panggil service dengan idPoli
    results, err := mc.Service.GetKaryawanTanpaShift(idShift, idRole, tanggal, idPoli)
    if err != nil {
        slog.Error("Failed to get karyawan without shift", "error", err)
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "gagal mengambil data karyawan: " + err.Error(),
            "data":    nil,
        })
    }

    // 6. Respons sukses
    slog.Info("Successfully retrieved karyawan without shift", "count", len(results))
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "berhasil mengambil data karyawan",
        "data":    results,
    })
}


// AssignShiftHandlerNew menerima query parameter id_poli, id_shift, tanggal,
// dan request body berisi array dari AssignShiftRequest.
// id_management diambil dari JWT.
func (sc *ShiftController) AssignShiftHandlerNew(c echo.Context) error {
	// Ambil query parameter
	idPoliStr := c.QueryParam("id_poli")
	idShiftStr := c.QueryParam("id_shift")
	tanggalStr := c.QueryParam("tanggal")
	if idPoliStr == "" || idShiftStr == "" || tanggalStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli, id_shift, dan tanggal harus disediakan",
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
	idShift, err := strconv.Atoi(idShiftStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_shift harus berupa angka",
			"data":    nil,
		})
	}

	// Validasi format tanggal DD/MM/YYYY
	if _, err := time.Parse("02/01/2006", tanggalStr); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "tanggal harus dalam format DD/MM/YYYY",
			"data":    nil,
		})
	}

	// Ambil data dari request body: array dari AssignShiftRequest
	var requests []services.AssignShiftRequest
	if err := c.Bind(&requests); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "invalid request payload: " + err.Error(),
			"data":    nil,
		})
	}
	if len(requests) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "request body tidak boleh kosong",
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
	idManagement :=claims.IDKaryawan
	if idManagement <= 0 {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "invalid management id in token",
			"data":    nil,
		})
	}

	// Panggil fungsi service untuk assign shift
	err = sc.Service.AssignShiftNew(idPoli, idShift, idManagement, tanggalStr, requests)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "failed to assign shift: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "shift berhasil ditambahkan",
		"data":    nil,
	})
}

func (sc *ShiftController) GetJadwalShiftHandler(c echo.Context) error {
	//-------------------------------- query-param -------------------------------
	idKaryawanStr := c.QueryParam("id_karyawan")
	if idKaryawanStr == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "id_karyawan is required",
			"data":    nil,
		})
	}
	idKaryawan, err := strconv.Atoi(idKaryawanStr)
	if err != nil || idKaryawan <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "invalid id_karyawan (must be positive integer)",
			"data":    nil,
		})
	}

	// bulan & tahun optional
	nowJakarta := time.Now().In(time.FixedZone("WIB", 7*3600))
	month := int(nowJakarta.Month())
	year  := nowJakarta.Year()

	if m := c.QueryParam("bulan"); m != "" {
		if v, err := strconv.Atoi(m); err == nil && v >= 1 && v <= 12 {
			month = v
		} else {
			return c.JSON(http.StatusBadRequest, echo.Map{
				"status":  http.StatusBadRequest,
				"message": "invalid bulan (1-12)",
				"data":    nil,
			})
		}
	}
	if y := c.QueryParam("tahun"); y != "" {
		if v, err := strconv.Atoi(y); err == nil && v >= 1900 {
			year = v
		} else {
			return c.JSON(http.StatusBadRequest, echo.Map{
				"status":  http.StatusBadRequest,
				"message": "invalid tahun",
				"data":    nil,
			})
		}
	}

	//-------------------------------- service call ------------------------------
	result, err := sc.Service.GetJadwalShift(idKaryawan, month, year)
	if err != nil {
		switch err {
		case services.ErrNoShiftData:
			return c.JSON(http.StatusNotFound, echo.Map{
				"status":  http.StatusNotFound,
				"message": "Tidak ada data shift untuk periode tersebut",
				"data":    nil,
			})
		default:
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"status":  http.StatusInternalServerError,
				"message": "Failed to fetch shift: " + err.Error(),
				"data":    nil,
			})
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  http.StatusOK,
		"message": "Success",
		"data":    result,
	})
}
