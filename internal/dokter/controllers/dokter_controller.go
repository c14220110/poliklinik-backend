package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	//"time"

	"github.com/c14220110/poliklinik-backend/internal/dokter/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

type LoginDokterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IDPoli   int    `json:"id_poli"`
}

type DokterController struct {
	Service *services.DokterService
}

func NewDokterController(service *services.DokterService) *DokterController {
	return &DokterController{Service: service}
}

func (dc *DokterController) LoginDokter(w http.ResponseWriter, r *http.Request) {
	var req LoginDokterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}
	if req.Username == "" || req.Password == "" || req.IDPoli == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Username, Password, and IDPoli are required",
			"data":    nil,
		})
		return
	}

	// Gunakan dc.Service.DB (field DB di dalam service) untuk query
	var idKaryawan int
	var nama, username, hashedPassword string
	query := "SELECT ID_Karyawan, Nama, Username, Password FROM Karyawan WHERE Username = ?"
	err := dc.Service.DB.QueryRow(query, req.Username).Scan(&idKaryawan, &nama, &username, &hashedPassword)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid username or password",
			"data":    nil,
		})
		return
	}

	// Verifikasi password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid username or password",
			"data":    nil,
		})
		return
	}

	// Cek role melalui Detail_Role_Karyawan dan Role (harus "Dokter")
	var roleName string
	roleQuery := `
		SELECT r.Nama_Role 
		FROM Detail_Role_Karyawan drk 
		JOIN Role r ON drk.ID_Role = r.ID_Role 
		WHERE drk.ID_Karyawan = ?
		LIMIT 1
	`
	err = dc.Service.DB.QueryRow(roleQuery, idKaryawan).Scan(&roleName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve role: " + err.Error(),
			"data":    nil,
		})
		return
	}
	if roleName != "Dokter" {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusForbidden,
			"message": "User is not a Dokter",
			"data":    nil,
		})
		return
	}

	// Cek shift aktif dari Shift_Karyawan dan Shift (dengan kondisi Tanggal = CURDATE())
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
	err = dc.Service.DB.QueryRow(shiftQuery, idKaryawan, req.IDPoli).Scan(&idShiftKaryawan, &jamMulai, &jamSelesai)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusUnauthorized,
				"message": "No active shift for this Dokter on the selected poliklinik",
				"data":    nil,
			})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve shift: " + err.Error(),
			"data":    nil,
		})
		return
	}

	extraClaims := map[string]interface{}{
		"role":       "Dokter",
		"privileges": []string{"assessment", "input_screening", "e_resep", "pulangkan_pasien"},
		"id_poli":    req.IDPoli,
	}
	token, err := utils.GenerateTokenWithClaims(idKaryawan, username, extraClaims)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to generate token: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Login successful",
		"data": map[string]interface{}{
			"id":           idKaryawan,
			"nama":         nama,
			"username":     username,
			"role":         "Dokter",
			"id_poli":      req.IDPoli,
			"token":        token,
			"shift": map[string]interface{}{
				"id_shift_karyawan": idShiftKaryawan,
				"jam_mulai":         jamMulai,
				"jam_selesai":       jamSelesai,
			},
		},
	})
}
