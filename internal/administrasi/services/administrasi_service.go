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

	// Query untuk mengambil data karyawan beserta role langsung dari tabel Role
	query := `
		SELECT k.id_karyawan, k.nama, k.username, k.password, k.created_at, r.nama_role
		FROM Karyawan k
		JOIN Role r ON k.id_role = r.id_role
		WHERE k.username = ?
	`
	err := s.DB.QueryRow(query, username).Scan(&admin.ID_Admin, &admin.Nama, &admin.Username, &admin.Password, &admin.CreatedAt, &roleName)
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

	return &admin, nil
}
