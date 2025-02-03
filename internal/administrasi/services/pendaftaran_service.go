package services

import (
	"database/sql"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
)

type PendaftaranService struct {
	DB *sql.DB
}

func NewPendaftaranService(db *sql.DB) *PendaftaranService {
	return &PendaftaranService{DB: db}
}

// DaftarPasien menyimpan data pasien baru ke tabel Pasien.
func (s *PendaftaranService) DaftarPasien(p models.Pasien) (int64, error) {
	query := `
		INSERT INTO Pasien 
      (Nama, Tanggal_Lahir, Jenis_Kelamin, Tempat_Lahir, Kelurahan, Kecamatan, Alamat, No_Telp, Poli_Tujuan, Tanggal_Registrasi)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.DB.Exec(
		query,
		p.Nama,
		p.TanggalLahir,
		p.JenisKelamin,
		p.TempatLahir,
		p.Kelurahan,
		p.Kecamatan,
		p.Alamat,
		p.NoTelp,
		p.PoliTujuan,
		time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}


// BuatAntrian membuat entri baru di tabel Antrian untuk pasien yang baru didaftarkan.
func (s *PendaftaranService) BuatAntrian(a models.Antrian) (int64, error) {
	query := `
		INSERT INTO Antrian (ID_Pasien, ID_Poli, Nomor_Antrian, Status, Created_At)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := s.DB.Exec(query, a.IDPasien, a.IDPoli, a.NomorAntrian, a.Status, time.Now())
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetListPasien mengambil data pasien beserta informasi antrian.
func (s *PendaftaranService) GetListPasien() ([]map[string]interface{}, error) {
	query := `
		SELECT p.ID_Pasien, p.Nama, a.Nomor_Antrian, p.Poli_Tujuan, a.Status
		FROM Pasien p
		LEFT JOIN Antrian a ON p.ID_Pasien = a.ID_Pasien
		ORDER BY p.Tanggal_Registrasi DESC
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var idPasien int
		var nama string
		var nomorAntrian sql.NullInt64
		var poliTujuan string
		var status sql.NullInt64

		if err := rows.Scan(&idPasien, &nama, &nomorAntrian, &poliTujuan, &status); err != nil {
			return nil, err
		}
		data := map[string]interface{}{
			"id_pasien":  idPasien,
			"nama":       nama,
			"poli_tujuan": poliTujuan,
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
