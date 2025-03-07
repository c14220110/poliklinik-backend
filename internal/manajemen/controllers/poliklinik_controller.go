package controllers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
)

type UpdatePoliRequest struct {
	NamaPoli   string `json:"nama_poli"`
	Keterangan string `json:"keterangan"`
	LogoPoli   string `json:"logo_poli"` // Untuk update, kirim path/logo baru (jika ada)
}

type PoliklinikController struct {
	Service *services.PoliklinikService
}

func NewPoliklinikController(service *services.PoliklinikService) *PoliklinikController {
	return &PoliklinikController{Service: service}
}

func (pc *PoliklinikController) GetPoliklinikList(c echo.Context) error {
	statusFilter := c.QueryParam("status")
	list, err := pc.Service.GetPoliklinikListFiltered(statusFilter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve poliklinik list: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Poliklinik list retrieved successfully",
		"data":    list,
	})
}

func (pc *PoliklinikController) SoftDeletePoliklinikHandler(c echo.Context) error {
	// Ambil id_poli dari query parameter
	idPoliStr := c.QueryParam("id_poli")
	if idPoliStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli query parameter is required",
			"data":    nil,
		})
	}
	idPoli, err := strconv.Atoi(idPoliStr)
	if err != nil || idPoli <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli must be a valid number",
			"data":    nil,
		})
	}

	// Ambil id_management dari JWT (dari token yang sudah diverifikasi oleh middleware)
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
	}
	idManagement, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idManagement <= 0 {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid management ID in token",
			"data":    nil,
		})
	}

	// Panggil service untuk soft delete poliklinik
	err = pc.Service.SoftDeletePoliklinik(idPoli, idManagement)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to soft delete poliklinik: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Poliklinik soft-deleted successfully",
		"data":    nil,
	})
}

// AddPoliklinikHandler menangani POST untuk menambah Poliklinik.
// Body berupa multipart form-data dengan field: nama_poli, keterangan, dan file logo_poli (opsional).
func (pc *PoliklinikController) AddPoliklinikHandler(c echo.Context) error {
	// Ambil nilai form
	namaPoli := c.FormValue("nama_poli")
	keterangan := c.FormValue("keterangan")
	if namaPoli == "" || keterangan == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "nama_poli and keterangan are required",
			"data":    nil,
		})
	}

	// Proses file upload (logo_poli)
	var logoPath string
	file, err := c.FormFile("logo_poli")
	if err == nil {
		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to open logo file: " + err.Error(),
				"data":    nil,
			})
		}
		defer src.Close()

		// Tentukan folder tujuan (misalnya "uploads/")
		uploadDir := "uploads"
		// Pastikan folder ada
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to create upload directory: " + err.Error(),
				"data":    nil,
			})
		}
		// Buat nama file unik (misalnya dengan menggunakan filepath.Base dan timestamp)
		dstPath := filepath.Join(uploadDir, file.Filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to create logo file: " + err.Error(),
				"data":    nil,
			})
		}
		defer dst.Close()
		if _, err = io.Copy(dst, src); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to save logo file: " + err.Error(),
				"data":    nil,
			})
		}
		// Simpan path file sebagai logo_poli
		logoPath = dstPath
	} else {
		// Jika tidak ada file, set logoPath kosong
		logoPath = ""
	}

	// Ambil id_management dari JWT (management yang melakukan penambahan)
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
	}
	idManagement, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idManagement <= 0 {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid management ID in token",
			"data":    nil,
		})
	}

	// Panggil service untuk menambah poliklinik dan mencatat di Management_Poli
	poliID, err := pc.Service.AddPoliklinikWithManagement(namaPoli, keterangan, logoPath, idManagement)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to add poliklinik: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Poliklinik added successfully",
		"data": map[string]interface{}{
			"id_poli": poliID,
		},
	})
}

// UpdatePoliklinikHandler menangani update data Poliklinik via multipart form-data.
// Query parameter: id_poli
// Form fields: nama_poli, keterangan, dan file (logo_poli) opsional.
func (pc *PoliklinikController) UpdatePoliklinikHandler(c echo.Context) error {
	// Ambil id_poli dari query parameter
	idPoliStr := c.QueryParam("id_poli")
	if idPoliStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli query parameter is required",
			"data":    nil,
		})
	}
	idPoli, err := strconv.Atoi(idPoliStr)
	if err != nil || idPoli <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli must be a valid number",
			"data":    nil,
		})
	}

	// Ambil nilai form fields
	namaPoli := c.FormValue("nama_poli")
	keterangan := c.FormValue("keterangan")
	if namaPoli == "" || keterangan == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "nama_poli and keterangan are required",
			"data":    nil,
		})
	}

	// Proses file upload untuk logo_poli jika ada
	var logoPath string
	file, err := c.FormFile("logo_poli")
	if err == nil {
		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to open logo file: " + err.Error(),
				"data":    nil,
			})
		}
		defer src.Close()

		uploadDir := "uploads"
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to create upload directory: " + err.Error(),
				"data":    nil,
			})
		}
		dstPath := filepath.Join(uploadDir, file.Filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to create logo file: " + err.Error(),
				"data":    nil,
			})
		}
		defer dst.Close()
		if _, err = io.Copy(dst, src); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to save logo file: " + err.Error(),
				"data":    nil,
			})
		}
		logoPath = dstPath
	} else {
		// Jika file tidak diupload, logoPath tetap kosong (tidak akan diupdate)
		logoPath = ""
	}

	// Ambil id_management dari JWT (management yang melakukan update)
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
	}
	idManagement, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idManagement <= 0 {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid management ID in token",
			"data":    nil,
		})
	}

	// Panggil service untuk update poliklinik
	err = pc.Service.UpdatePoliklinikWithOptionalLogo(idPoli, namaPoli, keterangan, logoPath, idManagement)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to update poliklinik: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Poliklinik updated successfully",
		"data": map[string]interface{}{
			"id_poli": idPoli,
		},
	})
}

func (pc *PoliklinikController) GetActivePoliklinikList(c echo.Context) error {
	list, err := pc.Service.GetActivePoliklinikList()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve active poliklinik list: " + err.Error(),
			"data":    nil,
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Active poliklinik list retrieved successfully",
		"data":    list,
	})
}

func (fc *PoliklinikController) GetFile(c echo.Context) error {
    filename := c.Param("filename")
    // Pastikan filename aman (misalnya, dengan validasi atau menggunakan filepath.Clean)
    filePath := filepath.Join("uploads", filename)
    return c.File(filePath)
}
