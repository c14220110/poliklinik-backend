package services

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
)

type AdministrasiService struct {
	DB *sql.DB
}

func NewAdministrasiService(db *sql.DB) *AdministrasiService {
	return &AdministrasiService{DB: db}
}

func (s *AdministrasiService) AuthenticateAdmin(username, password string) (*models.Administrasi, error) {
	var admin models.Administrasi
	var roleName string
	var roleID int

	// Query diperbarui untuk join dengan Detail_Role_Karyawan dan Role
	query := `
		SELECT k.id_karyawan, k.nama, k.username, k.password, k.created_at, drk.id_role, r.nama_role
		FROM Karyawan k
		JOIN Detail_Role_Karyawan drk ON k.id_karyawan = drk.id_karyawan
		JOIN Role r ON drk.id_role = r.id_role
		WHERE k.username = ?
	`
	err := s.DB.QueryRow(query, username).Scan(&admin.ID_Admin, &admin.Nama, &admin.Username, &admin.Password, &admin.CreatedAt, &roleID, &roleName)
	if err != nil {
		return nil, err
	}

	// Verifikasi password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Pastikan role adalah "Administrasi"
	if roleName != "Administrasi" {
		return nil, errors.New("user does not have administrasi privileges")
	}

	admin.ID_Role = roleID

	// Ambil daftar privilege yang dimiliki karyawan
	rows, err := s.DB.Query("SELECT id_privilege FROM Detail_Privilege_Karyawan WHERE id_karyawan = ?", admin.ID_Admin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var privileges []int
	for rows.Next() {
		var priv int
		if err := rows.Scan(&priv); err != nil {
			return nil, err
		}
		privileges = append(privileges, priv)
	}
	admin.Privileges = privileges

	return &admin, nil
}
