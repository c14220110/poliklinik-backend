package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
)

type AddKaryawanRequest struct {
	NIK          string `json:"nik"`
	Nama         string `json:"nama"`
	JenisKelamin string `json:"jenis_kelamin"`  // ditambahkan
	TanggalLahir string `json:"tanggal_lahir"`
	Alamat       string `json:"alamat"`
	NoTelp       string `json:"no_telp"`
	Role         string `json:"role"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}


type UpdateKaryawanRequest struct {
	// Tidak perlu menyertakan id_karyawan di body, karena diambil dari query
	NIK          string `json:"nik"`
	Nama         string `json:"nama"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	TanggalLahir string `json:"tanggal_lahir"`
	Alamat       string `json:"alamat"`
	NoTelp       string `json:"no_telp"`
	Role         string `json:"role"`
}

type KaryawanController struct {
    Service *services.ManagementService
}

func NewKaryawanController(service *services.ManagementService) *KaryawanController {
    return &KaryawanController{Service: service}
}

func (kc *KaryawanController) AddKaryawan(c echo.Context) error {
	var req AddKaryawanRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
	}

	// Parse tanggal lahir
	parsedDate, err := time.Parse("2006-01-02", req.TanggalLahir)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid date format for tanggal_lahir. Use YYYY-MM-DD",
			"data":    nil,
		})
	}

	// Buat objek Karyawan, termasuk field jenis_kelamin
	karyawan := models.Karyawan{
		NIK:          req.NIK,
		Nama:         req.Nama,
		JenisKelamin: req.JenisKelamin,
		TanggalLahir: parsedDate,
		Alamat:       req.Alamat,
		NoTelp:       req.NoTelp,
		Username:     req.Username,
		Password:     req.Password,
	}

	// Ambil klaim JWT dari context
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
	}

	// Konversi id_management dari JWT ke integer
	idManagement, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idManagement <= 0 {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid management ID in token",
			"data":    nil,
		})
	}

	// Panggil service untuk menambahkan karyawan dengan created_by dan updated_by sebagai idManagement
	idKaryawan, err := kc.Service.AddKaryawan(karyawan, req.Role, idManagement, idManagement, idManagement)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to add karyawan: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Karyawan added successfully",
		"data": map[string]interface{}{
			"id_karyawan": idKaryawan,
		},
	})
}




func (kc *KaryawanController) GetKaryawanListHandler(c echo.Context) error {
	idRole := c.QueryParam("id_role")
	status := c.QueryParam("status")

	list, err := kc.Service.GetKaryawanListFiltered(idRole, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve karyawan list: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Karyawan list retrieved successfully",
		"data":    list,
	})
}



func (kc *KaryawanController) UpdateKaryawanHandler(c echo.Context) error {
	// Ambil id_karyawan dari query parameter
	idKaryawanStr := c.QueryParam("id_karyawan")
	if idKaryawanStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_karyawan query parameter is required",
			"data":    nil,
		})
	}
	idKaryawanInt, err := strconv.Atoi(idKaryawanStr)
	if err != nil || idKaryawanInt <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_karyawan must be a valid number",
			"data":    nil,
		})
	}
	// Konversi ke int64
	idKaryawan := int64(idKaryawanInt)

	var req UpdateKaryawanRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
	}

	// Validasi minimal field
	if req.NIK == "" || req.Nama == "" || req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "nik, nama, username, and password are required",
			"data":    nil,
		})
	}

	parsedDate, err := time.Parse("2006-01-02", req.TanggalLahir)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid date format for tanggal_lahir. Use YYYY-MM-DD",
			"data":    nil,
		})
	}

	// Buat objek Karyawan untuk update; IDKaryawan diisi dari query parameter yang telah dikonversi
	karyawan := models.Karyawan{
		IDKaryawan:   idKaryawan,
		NIK:          req.NIK,
		Nama:         req.Nama,
		Username:     req.Username,
		Password:     req.Password,
		TanggalLahir: parsedDate,
		Alamat:       req.Alamat,
		NoTelp:       req.NoTelp,
	}

	// Ambil klaim JWT dari context
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
	}

	// Konversi id_management dari JWT ke integer
	idManagementInt, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idManagementInt <= 0 {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid management ID in token",
			"data":    nil,
		})
	}
	idManagement := idManagementInt

	// Panggil service untuk update karyawan
	updatedID, err := kc.Service.UpdateKaryawan(karyawan, req.Role, idManagement)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to update karyawan: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Karyawan updated successfully",
		"data": map[string]interface{}{
			"id_karyawan": updatedID,
		},
	})
}




func (kc *KaryawanController) SoftDeleteKaryawanHandler(c echo.Context) error {
    idStr := c.QueryParam("id_karyawan")
    if idStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_karyawan parameter is required",
            "data":    nil,
        })
    }

    idKaryawan, err := strconv.Atoi(idStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_karyawan must be a number",
            "data":    nil,
        })
    }

    // Ambil klaim JWT dari context
    claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
    if !ok || claims == nil {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid or missing token claims",
            "data":    nil,
        })
    }

    err = kc.Service.SoftDeleteKaryawan(idKaryawan, claims.Username)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to soft delete karyawan: " + err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Karyawan soft-deleted successfully",
        "data":    nil,
    })
}

func (kc *KaryawanController) AddPrivilegeHandler(c echo.Context) error {
	// Ambil id_karyawan dari query parameter
	idKaryawanStr := c.QueryParam("id_karyawan")
	if idKaryawanStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_karyawan harus disediakan",
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
	
	// Parsing request body berupa array privilege
	var req struct {
		Privileges []int `json:"privileges"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request body: " + err.Error(),
			"data":    nil,
		})
	}
	if len(req.Privileges) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Setidaknya satu privilege harus disediakan",
			"data":    nil,
		})
	}
	
	// Panggil service untuk menambahkan privilege
	err = kc.Service.AddPrivilegesToKaryawan(idKaryawan, req.Privileges)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to add privileges: " + err.Error(),
			"data":    nil,
		})
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Privileges added successfully",
		"data":    nil,
	})
}
