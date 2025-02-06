package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
)

type PendaftaranService struct {
	DB *sql.DB
}

func NewPendaftaranService(db *sql.DB) *PendaftaranService {
	return &PendaftaranService{DB: db}
}

func (s *PendaftaranService) GetListPasien() ([]map[string]interface{}, error) {
	query := `
		SELECT 
			p.Nama, 
			rm.ID_RM, 
			pl.Nama_Poli, 
			a.Nomor_Antrian, 
			a.Status
		FROM Pasien p
		LEFT JOIN Rekam_Medis rm ON p.ID_Pasien = rm.ID_Pasien
		LEFT JOIN Antrian a ON p.ID_Pasien = a.ID_Pasien
		LEFT JOIN Poliklinik pl ON a.ID_Poli = pl.ID_Poli
		ORDER BY p.Tanggal_Registrasi DESC
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var nama string
		var idRM sql.NullInt64
		var namaPoli sql.NullString
		var nomorAntrian sql.NullInt64
		var status sql.NullInt64

		if err := rows.Scan(&nama, &idRM, &namaPoli, &nomorAntrian, &status); err != nil {
			return nil, err
		}

		data := map[string]interface{}{
			"nama":          nama,
			"id_rm":         nil,
			"nama_poli":     nil,
			"nomor_antrian": nil,
			"status":        nil,
		}

		if idRM.Valid {
			data["id_rm"] = idRM.Int64
		}
		if namaPoli.Valid {
			data["nama_poli"] = namaPoli.String
		}
		if nomorAntrian.Valid {
			data["nomor_antrian"] = nomorAntrian.Int64
		}
		if status.Valid {
			data["status"] = status.Int64
		}

		result = append(result, data)
	}
	return result, nil
}


func (s *PendaftaranService) RegisterPasienWithAntrian(p models.Pasien, a models.Antrian) (int64, int64, error) {
	// Cek apakah NIK sudah ada di database
	var existingID int
	err := s.DB.QueryRow("SELECT ID_Pasien FROM Pasien WHERE NIK = ?", p.Nik).Scan(&existingID)
	if err == nil {
		// Jika NIK sudah ditemukan, kembalikan error khusus
		return 0, 0, fmt.Errorf("NIK sudah terdaftar")
	} else if err != sql.ErrNoRows {
		// Jika error lain selain "tidak ditemukan", kembalikan error
		return 0, 0, err
	}

	// Mulai transaksi
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, 0, err
	}

	// Insert data pasien (tanpa data poli, karena itu akan disimpan di tabel Antrian)
	queryPasien := `
		INSERT INTO Pasien 
			(Nama, Tanggal_Lahir, Jenis_Kelamin, Tempat_Lahir, Kelurahan, Kecamatan, NIK, No_Telp, Alamat, Tanggal_Registrasi)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := tx.Exec(queryPasien,
		p.Nama,
		p.TanggalLahir,
		p.JenisKelamin,
		p.TempatLahir,
		p.Kelurahan,
		p.Kecamatan,
		p.Nik,
		p.NoTelp,
		p.Alamat,
		time.Now(),
	)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}
	patientID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// Hitung rentang waktu hari ini berdasarkan zona waktu aplikasi
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Hitung Nomor_Antrian untuk poli yang dipilih, hanya berdasarkan record yang dibuat hari ini.
	var maxNomor sql.NullInt64
	err = tx.QueryRow(
		"SELECT COALESCE(MAX(Nomor_Antrian), 0) FROM Antrian WHERE ID_Poli = ? AND Created_At >= ? AND Created_At < ?",
		a.IDPoli, startOfDay, endOfDay,
	).Scan(&maxNomor)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}
	nextNomor := int64(1)
	if maxNomor.Valid {
		nextNomor = maxNomor.Int64 + 1
	}

	// Insert data antrian dengan Nomor_Antrian yang telah dihitung dan status 0 (Menunggu)
	queryAntrian := `
		INSERT INTO Antrian 
			(ID_Pasien, ID_Poli, Nomor_Antrian, Status, Created_At)
		VALUES (?, ?, ?, ?, ?)
	`
	a.IDPasien = int(patientID)
	a.NomorAntrian = int(nextNomor)
	a.Status = 0
	_, err = tx.Exec(queryAntrian,
		a.IDPasien,
		a.IDPoli,
		a.NomorAntrian,
		a.Status,
		time.Now(),
	)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// Commit transaksi
	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}

	return patientID, nextNomor, nil
}
