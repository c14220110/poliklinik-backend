package services

import (
	"database/sql"
	"errors"

	"log/slog"

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

	// Log upaya login
	slog.Info("Attempting login", "username", username)

	// Query menggunakan join dengan Detail_Role_Karyawan dan Role
		query := `
		SELECT k.id_karyawan, k.nama, k.username, k.password, k.created_at, drk.id_role, r.nama_role
		FROM Karyawan k
		JOIN Detail_Role_Karyawan drk ON k.id_karyawan = drk.id_karyawan
		JOIN Role r ON drk.id_role = r.id_role
		WHERE k.username = ? AND r.nama_role = 'Administrasi'
	`
	err := s.DB.QueryRow(query, username).Scan(&admin.ID_Admin, &admin.Nama, &admin.Username, &admin.Password, &admin.CreatedAt, &roleID, &roleName)
	if err != nil {
		slog.Error("QueryRow error in AuthenticateAdmin", "username", username, "error", err)
		return nil, err
	}

	// Verifikasi password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		slog.Error("Password mismatch", "username", username)
		return nil, errors.New("invalid credentials")
	}

	// Pastikan role adalah "Administrasi"
	if roleName != "Administrasi" {
		slog.Error("User does not have administrasi privileges", "username", username, "role", roleName)
		return nil, errors.New("user does not have administrasi privileges")
	}

	admin.ID_Role = roleID

	// Ambil daftar privilege yang dimiliki oleh karyawan
	rows, err := s.DB.Query("SELECT id_privilege FROM Detail_Privilege_Karyawan WHERE id_karyawan = ?", admin.ID_Admin)
	if err != nil {
		slog.Error("Failed to query privileges", "username", username, "error", err)
		return nil, err
	}
	defer rows.Close()

	var privileges []int
	for rows.Next() {
		var priv int
		if err := rows.Scan(&priv); err != nil {
			slog.Error("Failed to scan privilege", "username", username, "error", err)
			return nil, err
		}
		privileges = append(privileges, priv)
	}
	admin.Privileges = privileges

	slog.Info("Login successful", "username", username, "id_karyawan", admin.ID_Admin)
	return &admin, nil
}
