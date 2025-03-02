package services

import (
	"database/sql"
	"fmt"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
)

type PrivilegeService struct {
	DB *sql.DB
}

func NewPrivilegeService(db *sql.DB) *PrivilegeService {
	return &PrivilegeService{DB: db}
}

// GetAllPrivileges mengambil semua data Privilege dari tabel Privilege
func (ps *PrivilegeService) GetAllPrivileges() ([]models.Privilege, error) {
	query := `
		SELECT 
			id_privilege, 
			nama_privilege, 
			deskripsi, 
			created_at, 
			updated_at, 
			deleted_at 
		FROM Privilege
		ORDER BY id_privilege
	`
	rows, err := ps.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var privileges []models.Privilege
	for rows.Next() {
		var p models.Privilege
		var deletedAt sql.NullTime
		if err := rows.Scan(&p.IDPrivilege, &p.NamaPrivilege, &p.Deskripsi, &p.CreatedAt, &p.UpdatedAt, &deletedAt); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}
		if deletedAt.Valid {
			p.DeletedAt = &deletedAt.Time
		} else {
			p.DeletedAt = nil
		}
		privileges = append(privileges, p)
	}

	return privileges, nil
}
