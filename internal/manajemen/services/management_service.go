package services

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
)

type ManagementService struct {
	DB *sql.DB
}

func NewManagementService(db *sql.DB) *ManagementService {
	return &ManagementService{DB: db}
}

// AuthenticateManagement memvalidasi login manajemen.
func (s *ManagementService) AuthenticateManagement(username, password string) (*models.Management, error) {
	var m models.Management
	query := "SELECT ID_Management, Username, Password, Nama FROM Management WHERE Username = ?"
	err := s.DB.QueryRow(query, username).Scan(&m.ID_Management, &m.Username, &m.Password, &m.Nama)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(m.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}
	return &m, nil
}
