package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	//"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
)

type RoleService struct {
	DB *sql.DB
}

func NewRoleService(db *sql.DB) *RoleService {
	return &RoleService{DB: db}
}

// AddRole memasukkan role baru ke tabel Role.
func (rs *RoleService) AddRole(namaRole string) (int64, error) {
	query := "INSERT INTO Role (nama_role, created_at, updated_at) VALUES (?, ?, ?)"
	now := time.Now()
	result, err := rs.DB.Exec(query, namaRole, now, now)
	if err != nil {
		return 0, fmt.Errorf("failed to add role: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get inserted role id: %v", err)
	}
	return id, nil
}

// UpdateRole memperbarui nama_role berdasarkan id_role.
func (rs *RoleService) UpdateRole(idRole int, namaRole string) error {
	query := "UPDATE Role SET nama_role = ?, updated_at = NOW() WHERE id_role = ?"
	_, err := rs.DB.Exec(query, namaRole, idRole)
	if err != nil {
		return fmt.Errorf("failed to update role: %v", err)
	}
	return nil
}

// SoftDeleteRole mengupdate deleted_at untuk melakukan soft delete.
func (rs *RoleService) SoftDeleteRole(idRole int) error {
	query := "UPDATE Role SET deleted_at = NOW() WHERE id_role = ?"
	_, err := rs.DB.Exec(query, idRole)
	if err != nil {
		return fmt.Errorf("failed to soft delete role: %v", err)
	}
	return nil
}

// GetRoleListFiltered mengambil daftar role dengan filter opsional status.
// Jika statusFilter kosong, kembalikan semua role.
func (rs *RoleService) GetRoleListFiltered(statusFilter string) ([]map[string]interface{}, error) {
	baseQuery := `
		SELECT id_role, nama_role, deleted_at
		FROM Role
	`
	conditions := []string{}
	params := []interface{}{}

	// Filter berdasarkan status
	if statusFilter != "" {
		statusLower := strings.ToLower(statusFilter)
		if statusLower == "aktif" {
			conditions = append(conditions, "deleted_at IS NULL")
		} else if statusLower == "nonaktif" {
			conditions = append(conditions, "deleted_at IS NOT NULL")
		}
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY id_role"

	rows, err := rs.DB.Query(baseQuery, params...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var idRole int
		var namaRole string
		var deletedAt sql.NullTime

		if err := rows.Scan(&idRole, &namaRole, &deletedAt); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}

		status := "aktif"
		if deletedAt.Valid {
			status = "nonaktif"
		}

		record := map[string]interface{}{
			"id_role":   idRole,
			"nama_role": namaRole,
			"status":    status,
		}
		list = append(list, record)
	}
	return list, nil
}

// ActivateRole mengubah deleted_at menjadi NULL untuk mengaktifkan kembali role.
func (rs *RoleService) ActivateRole(idRole int) error {
	query := "UPDATE Role SET deleted_at = NULL, updated_at = ? WHERE id_role = ?"
	_, err := rs.DB.Exec(query, time.Now(), idRole)
	if err != nil {
		return fmt.Errorf("failed to activate role: %v", err)
	}
	return nil
}

func (s *ManagementService) AddRolesToKaryawan(idKaryawan int, roles []int) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	// Pastikan rollback jika terjadi error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Loop setiap id_role yang ingin ditambahkan
	for _, roleID := range roles {
		// Cek apakah role sudah pernah di-assign ke karyawan tersebut
		var count int
		err = tx.QueryRow("SELECT COUNT(*) FROM Detail_Role_Karyawan WHERE id_role = ? AND id_karyawan = ?", roleID, idKaryawan).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			// Jika sudah ada, lewati (atau bisa juga mengembalikan error jika diperlukan)
			continue
		}

		// Insert record ke Detail_Role_Karyawan
		_, err = tx.Exec("INSERT INTO Detail_Role_Karyawan (id_role, id_karyawan) VALUES (?, ?)", roleID, idKaryawan)
		if err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}
