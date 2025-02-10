package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type UserController struct {
	DB *sql.DB
}

func NewUserController(db *sql.DB) *UserController {
	return &UserController{DB: db}
}

// GetKaryawanList mengambil daftar karyawan (ID_Karyawan dan Nama)
func (uc *UserController) GetKaryawanList(w http.ResponseWriter, r *http.Request) {
	query := "SELECT ID_Karyawan, Nama FROM Karyawan ORDER BY Nama ASC"
	rows, err := uc.DB.Query(query)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve karyawan list: " + err.Error(),
			"data":    nil,
		})
		return
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var id int
		var nama string
		if err := rows.Scan(&id, &nama); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to scan row: " + err.Error(),
				"data":    nil,
			})
			return
		}
		result = append(result, map[string]interface{}{
			"id_karyawan": id,
			"nama":        nama,
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Karyawan list retrieved successfully",
		"data":    result,
	})
}

// UpdateKaryawanRequest adalah struktur request untuk update karyawan.
type UpdateKaryawanRequest struct {
	ID_Karyawan int    `json:"id_karyawan"`
	Nama        string `json:"nama"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Role        string `json:"role"`
}

// UpdateKaryawan mengupdate data di tabel Karyawan dan update role di Detail_Role_Karyawan.
func (uc *UserController) UpdateKaryawan(w http.ResponseWriter, r *http.Request) {
	var req UpdateKaryawanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}
	if req.ID_Karyawan == 0 || req.Nama == "" || req.Username == "" || req.Role == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_karyawan, nama, username, and role are required",
			"data":    nil,
		})
		return
	}

	tx, err := uc.DB.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to begin transaction: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Update tabel Karyawan.
	var hashedPassword string
	if req.Password != "" {
		hp, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to hash password: " + err.Error(),
				"data":    nil,
			})
			return
		}
		hashedPassword = string(hp)
	} else {
		// Jika password tidak diupdate, ambil password lama.
		err = tx.QueryRow("SELECT Password FROM Karyawan WHERE ID_Karyawan = ?", req.ID_Karyawan).Scan(&hashedPassword)
		if err != nil {
			tx.Rollback()
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to retrieve existing password: " + err.Error(),
				"data":    nil,
			})
			return
		}
	}

	_, err = tx.Exec(`UPDATE Karyawan 
		SET Nama = ?, Username = ?, Password = ?, Updated_At = ? 
		WHERE ID_Karyawan = ?`,
		req.Nama, req.Username, hashedPassword, time.Now(), req.ID_Karyawan)
	if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to update Karyawan: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Update role karyawan: pertama, dapatkan ID_Role dari tabel Role berdasarkan nama role.
	var idRole int
	err = tx.QueryRow("SELECT ID_Role FROM Role WHERE Nama_Role = ?", req.Role).Scan(&idRole)
	if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve role id: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Hapus record lama di Detail_Role_Karyawan untuk karyawan ini.
	_, err = tx.Exec("DELETE FROM Detail_Role_Karyawan WHERE ID_Karyawan = ?", req.ID_Karyawan)
	if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to delete old role details: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Dapatkan data karyawan (NIK, Alamat, No_Telp) agar diinsert ke Detail_Role_Karyawan.
	var nik, alamat, noTelp string
	err = tx.QueryRow("SELECT NIK, Alamat, No_Telp FROM Karyawan WHERE ID_Karyawan = ?", req.ID_Karyawan).Scan(&nik, &alamat, &noTelp)
	if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve karyawan details: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Insert record baru di Detail_Role_Karyawan.
	_, err = tx.Exec(`INSERT INTO Detail_Role_Karyawan (ID_Role, ID_Karyawan, Nama, NIK, Alamat, No_Telp)
		VALUES (?, ?, ?, ?, ?, ?)`,
		idRole, req.ID_Karyawan, req.Nama, nik, alamat, noTelp)
	if err != nil {
		tx.Rollback()
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to insert new role details: " + err.Error(),
			"data":    nil,
		})
		return
	}

	if err = tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Transaction commit failed: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "Karyawan updated successfully",
		"data":    nil,
	})
}
