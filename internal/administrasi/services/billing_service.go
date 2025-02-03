package services

import (
	"database/sql"
	"time"
	//"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
)

type BillingService struct {
	DB *sql.DB
}

func NewBillingService(db *sql.DB) *BillingService {
	return &BillingService{DB: db}
}

// GetRecentBilling mengambil data billing terbaru dengan opsi filter status.
func (s *BillingService) GetRecentBilling(filterStatus *int) ([]map[string]interface{}, error) {
	query := `
		SELECT p.ID_Pasien, p.Nama, b.Status, b.Created_At
		FROM Pasien p
		JOIN Billing b ON p.ID_Pasien = b.ID_Pasien
	`
	var rows *sql.Rows
	var err error
	if filterStatus != nil {
		query += " WHERE b.Status = ? ORDER BY b.Created_At DESC"
		rows, err = s.DB.Query(query, *filterStatus)
	} else {
		query += " ORDER BY b.Created_At DESC"
		rows, err = s.DB.Query(query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var idPasien int
		var nama string
		var status int
		var createdAt time.Time

		if err := rows.Scan(&idPasien, &nama, &status, &createdAt); err != nil {
			return nil, err
		}
		data := map[string]interface{}{
			"id_pasien":  idPasien,
			"nama":       nama,
			"status":     status,
			"created_at": createdAt,
		}
		result = append(result, data)
	}
	return result, nil
}

// GetBillingDetail mengambil detail billing untuk pasien tertentu.
func (s *BillingService) GetBillingDetail(idPasien int) (map[string]interface{}, error) {
	query := `
		SELECT p.Nama, rm.ID_RM, d.Nama, b.Status, b.Created_At
		FROM Billing b
		JOIN Pasien p ON b.ID_Pasien = p.ID_Pasien
		LEFT JOIN Rekam_Medis rm ON p.ID_Pasien = rm.ID_Pasien
		LEFT JOIN Dokter d ON rm.ID_Dokter = d.ID_Dokter
		WHERE p.ID_Pasien = ?
		LIMIT 1
	`
	row := s.DB.QueryRow(query, idPasien)
	var namaPasien, namaDokter string
	var idRM sql.NullInt64
	var status int
	var createdAt time.Time
	if err := row.Scan(&namaPasien, &idRM, &namaDokter, &status, &createdAt); err != nil {
		return nil, err
	}
	detail := map[string]interface{}{
		"nama_pasien": namaPasien,
		"nomor_rm":    idRM.Int64,
		"nama_dokter": namaDokter,
		"status":      status,
		"tanggal_jam": createdAt,
	}
	return detail, nil
}
