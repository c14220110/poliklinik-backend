package services

import (
	"database/sql"
	//"fmt"
	//"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
)

// PendaftaranService menangani registrasi pasien.
type PendaftaranService struct {
	DB *sql.DB
}

func NewPendaftaranService(db *sql.DB) *PendaftaranService {
	return &PendaftaranService{DB: db}
}

// RegisterPasienWithRekamMedisAndAntrian melakukan registrasi pasien, pembuatan rekam medis (jika belum ada),
// pembuatan riwayat kunjungan, dan pembuatan antrian, dalam satu transaksi.
// operatorID adalah ID_Karyawan yang menginput (diambil dari token JWT).
func (s *PendaftaranService) RegisterPasienWithRekamMedisAndAntrian(p models.Pasien, idPoli int, operatorID int) (int64, int64, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, 0, err
	}

	var patientID int64
	var idRM int64

	// 1. Cek apakah pasien sudah terdaftar berdasarkan NIK.
	var existingID int64
	err = tx.QueryRow("SELECT ID_Pasien FROM Pasien WHERE NIK = ?", p.NIK).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return 0, 0, err
	}

	if err == sql.ErrNoRows {
		// Pasien belum terdaftar: Insert ke tabel Pasien.
		queryPasien := `
			INSERT INTO Pasien 
				(Nama, Tanggal_Lahir, Jenis_Kelamin, Tempat_Lahir, NIK, Kelurahan, Kecamatan, Alamat, No_Telp)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		res, err := tx.Exec(queryPasien,
			p.Nama,
			p.TanggalLahir,
			p.JenisKelamin,
			p.TempatLahir,
			p.NIK,
			p.Kelurahan,
			p.Kecamatan,
			p.Alamat,
			p.NoTelp,
		)
		if err != nil {
			tx.Rollback()
			return 0, 0, err
		}
		patientID, err = res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return 0, 0, err
		}
	} else {
		// Pasien sudah ada.
		patientID = existingID
	}

	// 2. Buat atau ambil rekam medis untuk pasien.
	err = tx.QueryRow("SELECT ID_RM FROM Rekam_Medis WHERE ID_Pasien = ? LIMIT 1", patientID).Scan(&idRM)
	if err == sql.ErrNoRows {
		// Belum ada rekam medis, buat baru.
		queryRM := `
			INSERT INTO Rekam_Medis (ID_Pasien)
			VALUES (?)
		`
		res, err := tx.Exec(queryRM, patientID)
		if err != nil {
			tx.Rollback()
			return 0, 0, err
		}
		idRM, err = res.LastInsertId()
		if err != nil {
			tx.Rollback()
			return 0, 0, err
		}
	} else if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 3. Buat record di tabel Riwayat_Kunjungan.
	queryRK := `
		INSERT INTO Riwayat_Kunjungan (ID_RM, ID_Poli, Created_At)
		VALUES (?, ?, NOW())
	`
	_, err = tx.Exec(queryRK, idRM, idPoli)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 4. Hitung nomor antrian untuk poli yang dipilih pada hari ini.
	// Gunakan rentang waktu: DATE(Created_At) = CURDATE()
	var maxNomor sql.NullInt64
	queryMax := `
		SELECT COALESCE(MAX(Nomor_Antrian), 0)
		FROM Antrian
		WHERE ID_Poli = ? AND DATE(Created_At) = CURDATE()
	`
	err = tx.QueryRow(queryMax, idPoli).Scan(&maxNomor)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}
	nomorAntrian := int64(1)
	if maxNomor.Valid {
		nomorAntrian = maxNomor.Int64 + 1
	}

	// 5. Buat record di tabel Antrian.
	queryAntrian := `
		INSERT INTO Antrian (ID_Pasien, ID_Poli, Nomor_Antrian, Status, Created_At)
		VALUES (?, ?, ?, 0, NOW())
	`
	_, err = tx.Exec(queryAntrian, patientID, idPoli, nomorAntrian)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, 0, err
	}

	return patientID, nomorAntrian, nil
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
