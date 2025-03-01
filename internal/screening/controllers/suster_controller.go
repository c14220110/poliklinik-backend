package controllers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/screening/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
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

    // Ambil data karyawan dari tabel Karyawan menggunakan sc.Service.DB
    var idKaryawan int
    var nama, username, hashedPassword string
    query := "SELECT ID_Karyawan, Nama, Username, Password FROM Karyawan WHERE Username = ?"
    err := sc.Service.DB.QueryRow(query, req.Username).Scan(&idKaryawan, &nama, &username, &hashedPassword)
    if err != nil {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid username or password",
            "data":    nil,
        })
    }

    // Verifikasi password
    if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid username or password",
            "data":    nil,
        })
    }

    // Cek role melalui Detail_Role_Karyawan dan Role (harus "Suster")
    var roleName string
    roleQuery := `
        SELECT r.Nama_Role 
        FROM Detail_Role_Karyawan drk 
        JOIN Role r ON drk.ID_Role = r.ID_Role 
        WHERE drk.ID_Karyawan = ?
        LIMIT 1
    `
    err = sc.Service.DB.QueryRow(roleQuery, idKaryawan).Scan(&roleName)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve role: " + err.Error(),
            "data":    nil,
        })
    }
    if roleName != "Suster" {
        return c.JSON(http.StatusForbidden, map[string]interface{}{
            "status":  http.StatusForbidden,
            "message": "User is not a Suster",
            "data":    nil,
        })
    }

    // Cek shift aktif dengan join Shift_Karyawan dan Shift
    var idShiftKaryawan int
    var jamMulai, jamSelesai string
    shiftQuery := `
        SELECT sk.ID_Shift_Karyawan, TIME(s.Jam_Mulai), TIME(s.Jam_Selesai)
        FROM Shift_Karyawan sk
        JOIN Shift s ON sk.ID_Shift = s.ID_Shift
        WHERE sk.ID_Karyawan = ? 
          AND sk.ID_Poli = ? 
          AND sk.Tanggal = CURDATE()
          AND (
             (s.Jam_Mulai < s.Jam_Selesai AND CURTIME() BETWEEN s.Jam_Mulai AND s.Jam_Selesai)
             OR (s.Jam_Mulai > s.Jam_Selesai AND (CURTIME() >= s.Jam_Mulai OR CURTIME() < s.Jam_Selesai))
          )
        LIMIT 1
    `
    err = sc.Service.DB.QueryRow(shiftQuery, idKaryawan, req.IDPoli).Scan(&idShiftKaryawan, &jamMulai, &jamSelesai)
    if err != nil {
        if err == sql.ErrNoRows {
            return c.JSON(http.StatusUnauthorized, map[string]interface{}{
                "status":  http.StatusUnauthorized,
                "message": "No active shift for this Suster on the selected poliklinik",
                "data":    nil,
            })
        }
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve shift: " + err.Error(),
            "data":    nil,
        })
    }

    // Generate token menggunakan fungsi JWT terpadu
    token, err := utils.GenerateJWTToken(strconv.Itoa(idKaryawan), "Suster", []map[string]interface{}{
        {"privilege": "input_screening"},
    }, req.IDPoli, username)
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
            "id":       idKaryawan,
            "nama":     nama,
            "username": username,
            "role":     "Suster",
            "id_poli":  req.IDPoli,
            "token":    token,
            "shift": map[string]interface{}{
                "id_shift_karyawan": idShiftKaryawan,
                "jam_mulai":         jamMulai,
                "jam_selesai":       jamSelesai,
            },
        },
    })
}