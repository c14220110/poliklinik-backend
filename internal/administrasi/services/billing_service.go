package services

import (
	"database/sql"
	"time"
	//"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
)

// BillingService menangani logika bisnis untuk data Billing.
type BillingService struct {
	DB *sql.DB
}

func NewBillingService(db *sql.DB) *BillingService {
	return &BillingService{DB: db}
}

// GetRecentBilling mengambil data billing dengan kolom:
// - nama (Pasien.Nama)
// - id_rm (Rekam_Medis.ID_RM)
// - nama_poli (Poliklinik.Nama_Poli, diambil dari join tabel Antrian)
// - status (Billing.Status)
// - id_billing (Billing.ID_Billing)
// Jika filterStatus tidak nil, query akan membatasi data billing berdasarkan status.
func (s *BillingService) GetRecentBilling(filterStatus *int) ([]map[string]interface{}, error) {
	// Query untuk join antara Billing, Pasien, Rekam_Medis, Antrian, dan Poliklinik.
	// Perhatikan: karena tabel Billing tidak langsung memiliki ID_Poli, kita join dengan tabel Antrian.
	query := `
		SELECT 
			p.Nama, 
			rm.ID_RM, 
			pl.Nama_Poli, 
			b.Status, 
			b.ID_Billing
		FROM Billing b
		JOIN Pasien p ON b.ID_Pasien = p.ID_Pasien
		LEFT JOIN Rekam_Medis rm ON p.ID_Pasien = rm.ID_Pasien
		LEFT JOIN Antrian a ON p.ID_Pasien = a.ID_Pasien
		LEFT JOIN Poliklinik pl ON a.ID_Poli = pl.ID_Poli
	`
	// Jika ada filter berdasarkan status billing, tambahkan klausa WHERE.
	if filterStatus != nil {
		query += " WHERE b.Status = ?"
	}
	query += " ORDER BY b.Created_At DESC"

	var rows *sql.Rows
	var err error
	if filterStatus != nil {
		rows, err = s.DB.Query(query, *filterStatus)
	} else {
		rows, err = s.DB.Query(query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var nama string
		var idRM sql.NullInt64
		var namaPoli sql.NullString
		var status int
		var idBilling int

		if err := rows.Scan(&nama, &idRM, &namaPoli, &status, &idBilling); err != nil {
			return nil, err
		}

		record := map[string]interface{}{
			"nama":       nama,
			"id_rm":      nil,
			"nama_poli":  nil,
			"status":     status,
			"id_billing": idBilling,
		}

		if idRM.Valid {
			record["id_rm"] = idRM.Int64
		}
		if namaPoli.Valid {
			record["nama_poli"] = namaPoli.String
		}

		result = append(result, record)
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
