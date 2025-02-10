package services

import (
	"database/sql"
	"errors"
	"fmt"

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
	query := "SELECT ID_Karyawan, Nama, Username, Password, Created_At FROM Karyawan WHERE Username = ?"
	err := s.DB.QueryRow(query, username).Scan(&admin.ID_Admin, &admin.Nama, &admin.Username, &admin.Password, &admin.CreatedAt)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	var roleName string
	roleQuery := `
		SELECT r.Nama_Role 
		FROM Detail_Role_Karyawan drk 
		JOIN Role r ON drk.ID_Role = r.ID_Role 
		WHERE drk.ID_Karyawan = ?
		LIMIT 1
	`
	err = s.DB.QueryRow(roleQuery, admin.ID_Admin).Scan(&roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve role: %v", err)
	}
	if roleName != "Administrasi" {
		return nil, errors.New("user does not have administrasi privileges")
	}

	return &admin, nil
}
