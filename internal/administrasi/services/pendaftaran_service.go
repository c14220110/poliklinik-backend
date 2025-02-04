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

// GetListPasien mengambil data pasien beserta informasi antrian.
func (s *PendaftaranService) GetListPasien() ([]map[string]interface{}, error) {
	query := `
		SELECT 
			p.ID_Pasien, 
			p.Nama, 
			rm.ID_RM, 
			p.Poli_Tujuan, 
			a.Nomor_Antrian, 
			a.Status
		FROM Pasien p
		LEFT JOIN Rekam_Medis rm ON p.ID_Pasien = rm.ID_Pasien
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
		var idRM sql.NullInt64
		var poliTujuan string
		var nomorAntrian sql.NullInt64
		var status sql.NullInt64

		if err := rows.Scan(&idPasien, &nama, &idRM, &poliTujuan, &nomorAntrian, &status); err != nil {
			return nil, err
		}

		// Konversi status ke string
		var statusString string
		if status.Valid {
			switch status.Int64 {
			case 0:
				statusString = "Menunggu"
			case 1:
				statusString = "Konsultasi"
			case 2:
				statusString = "Selesai"
			default:
				statusString = "Tidak Diketahui"
			}
		} else {
			statusString = "Belum Terdaftar"
		}

		// Masukkan data ke dalam map
		data := map[string]interface{}{
			"id_pasien":    idPasien,
			"nama":         nama,
			"id_rm":        nil, // Default null
			"poli_tujuan":  poliTujuan,
			"nomor_antrian": nil,
			"status":       statusString, // Kirim status dalam bentuk string
		}

		if idRM.Valid {
			data["id_rm"] = idRM.Int64
		}
		if nomorAntrian.Valid {
			data["nomor_antrian"] = nomorAntrian.Int64
		}

		result = append(result, data)
	}
	return result, nil
}

// RegisterPasienWithAntrian melakukan pendaftaran pasien dan pembuatan antrian
// dalam satu transaksi. Fungsi ini menghitung Nomor_Antrian dengan mengambil nilai maksimum
// nomor antrian pada poli yang sama dan menambahkan 1.
// Mengembalikan: patientID, nomorAntrian, dan error (jika ada).
func (s *PendaftaranService) RegisterPasienWithAntrian(p models.Pasien, a models.Antrian) (int64, int64, error) {
	// Mulai transaksi
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, 0, err
	}

	// Insert data pasien
	queryPasien := `
		INSERT INTO Pasien 
			(Nama, Tanggal_Lahir, Jenis_Kelamin, Tempat_Lahir, Kelurahan, Kecamatan, Alamat, No_Telp, Poli_Tujuan, Tanggal_Registrasi)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := tx.Exec(queryPasien,
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
		tx.Rollback()
		return 0, 0, err
	}
	patientID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// Hitung Nomor_Antrian untuk poli yang dipilih
	var maxNomor sql.NullInt64
	err = tx.QueryRow("SELECT COALESCE(MAX(Nomor_Antrian), 0) FROM Antrian WHERE ID_Poli = ?", a.IDPoli).Scan(&maxNomor)
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

