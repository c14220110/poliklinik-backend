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

// RegisterPasienWithKunjungan melakukan registrasi pasien baru beserta:
// - Insert data Pasien
// - Pembuatan Rekam Medis (dengan ID_Pasien)
// - Pencatatan Riwayat Kunjungan (dengan ID_RM)
// - Hubungan Kunjungan dengan Poli (di tabel Kunjungan_Poli)
// - Pembuatan nomor antrian di tabel Antrian (nomor antrian unik per poli per hari)
func (s *PendaftaranService) RegisterPasienWithKunjungan(p models.Pasien, idPoli int, operatorID int) (int64, int64, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, 0, err
	}

	// 1. Cek apakah NIK sudah ada di tabel Pasien
	var existingID int
	err = tx.QueryRow("SELECT ID_Pasien FROM Pasien WHERE NIK = ?", p.NIK).Scan(&existingID)
	if err == nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("NIK sudah terdaftar")
	} else if err != sql.ErrNoRows {
		tx.Rollback()
		return 0, 0, err
	}

	// 2. Insert data pasien ke tabel Pasien
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
	patientID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 3. Pembuatan Rekam Medis (hanya masukkan ID_Pasien)
	queryRM := `
		INSERT INTO Rekam_Medis (ID_Pasien)
		VALUES (?)
	`
	_, err = tx.Exec(queryRM, patientID)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}
	// ID_RM dihasilkan otomatis, tapi tidak dipakai lebih lanjut di sini

	// 4. Pencatatan Riwayat Kunjungan (insert hanya ID_RM)
	// Dapatkan ID_RM dari rekam medis yang baru saja dibuat
	var idRM int64
	err = tx.QueryRow("SELECT ID_RM FROM Rekam_Medis WHERE ID_Pasien = ? ORDER BY Created_At DESC LIMIT 1", patientID).Scan(&idRM)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	queryRK := `
		INSERT INTO Riwayat_Kunjungan (ID_RM)
		VALUES (?)
	`
	_, err = tx.Exec(queryRK, idRM)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 5. Hubungan Kunjungan dengan Poli: Insert ke tabel Kunjungan_Poli
	// Asumsikan tabel Kunjungan_Poli sudah ada dengan kolom (ID_Poli, ID_Kunjungan)
	queryKP := `
		INSERT INTO Kunjungan_Poli (ID_Poli, ID_Kunjungan)
		VALUES (?, (SELECT ID_Kunjungan FROM Riwayat_Kunjungan WHERE ID_RM = ? ORDER BY Created_At DESC LIMIT 1))
	`
	_, err = tx.Exec(queryKP, idPoli, idRM)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 6. Pembuatan Nomor Antrian:
	// Hitung nomor antrian untuk poli yang dipilih pada hari ini (reset tiap hari)
	today := time.Now().Format("2006-01-02")
	var maxNomor sql.NullInt64
	queryMax := `
		SELECT COALESCE(MAX(Nomor_Antrian), 0)
		FROM Antrian
		WHERE ID_Poli = ? AND DATE(Created_At) = ?
	`
	err = tx.QueryRow(queryMax, idPoli, today).Scan(&maxNomor)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}
	nomorAntrian := int64(1)
	if maxNomor.Valid && maxNomor.Int64 > 0 {
		nomorAntrian = maxNomor.Int64 + 1
	}

	// 7. Insert data antrian ke tabel Antrian
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


// UpdateKunjunganPasien mencari pasien berdasarkan NIK, mengupdate data pasien (misalnya Nama dan No_Telp),
// lalu mencatat kunjungan tambahan dengan membuat record baru di Riwayat_Kunjungan, Kunjungan_Poli, dan Antrian.
// idPoli: poliklinik yang dipilih
func (s *PendaftaranService) UpdateKunjunganPasien(nik string, updated models.Pasien, idPoli int) (int64, int64, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, 0, err
	}

	// 1. Cari pasien berdasarkan NIK
	var patientID int64
	queryPatient := "SELECT ID_Pasien FROM Pasien WHERE NIK = ?"
	err = tx.QueryRow(queryPatient, nik).Scan(&patientID)
	if err != nil {
		if err == sql.ErrNoRows {
			tx.Rollback()
			return 0, 0, fmt.Errorf("pasien dengan nik %s tidak ditemukan", nik)
		}
		tx.Rollback()
		return 0, 0, err
	}

	// 2. Update data pasien (misalnya, Nama dan No_Telp)
	queryUpdate := "UPDATE Pasien SET Nama = ?, No_Telp = ? WHERE ID_Pasien = ?"
	_, err = tx.Exec(queryUpdate, updated.Nama, updated.NoTelp, patientID)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 3. Ambil Rekam_Medis untuk pasien tersebut
	var idRM int64
	queryRM := "SELECT ID_RM FROM Rekam_Medis WHERE ID_Pasien = ? ORDER BY Created_At DESC LIMIT 1"
	err = tx.QueryRow(queryRM, patientID).Scan(&idRM)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 4. Insert record baru di Riwayat_Kunjungan
	queryRK := "INSERT INTO Riwayat_Kunjungan (ID_RM) VALUES (?)"
	res, err := tx.Exec(queryRK, idRM)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}
	// Ambil ID_Kunjungan baru (meskipun nanti hanya digunakan untuk relasi)
	idKunjungan, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 5. Insert record di Kunjungan_Poli untuk mengaitkan Riwayat_Kunjungan dengan Poliklinik
	queryKP := "INSERT INTO Kunjungan_Poli (ID_Poli, ID_Kunjungan) VALUES (?, ?)"
	_, err = tx.Exec(queryKP, idPoli, idKunjungan)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 6. Hitung nomor antrian untuk poli yang bersangkutan pada hari ini
	today := time.Now().Format("2006-01-02")
	var maxNomor sql.NullInt64
	queryMax := `
		SELECT COALESCE(MAX(Nomor_Antrian), 0)
		FROM Antrian
		WHERE ID_Poli = ? AND DATE(Created_At) = ?
	`
	err = tx.QueryRow(queryMax, idPoli, today).Scan(&maxNomor)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}
	nomorAntrian := int64(1)
	if maxNomor.Valid && maxNomor.Int64 > 0 {
		nomorAntrian = maxNomor.Int64 + 1
	}

	// 7. Insert data antrian ke tabel Antrian
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

func (s *PendaftaranService) GetAllPasienData() ([]map[string]interface{}, error) {
	query := `
		SELECT ID_Pasien, Nama, Tanggal_Lahir, Jenis_Kelamin, Tempat_Lahir, NIK, Kelurahan, Kecamatan, Alamat, No_Telp, Tanggal_Registrasi
		FROM Pasien
		ORDER BY Tanggal_Registrasi DESC
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var idPasien int
		var nama string
		var tanggalLahir string // atau time.Time, jika ingin
		var jenisKelamin, tempatLahir, nik, kelurahan, kecamatan, alamat, noTelp string
		var tanggalRegistrasi string
		err = rows.Scan(&idPasien, &nama, &tanggalLahir, &jenisKelamin, &tempatLahir, &nik, &kelurahan, &kecamatan, &alamat, &noTelp, &tanggalRegistrasi)
		if err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"ID_Pasien":           idPasien,
			"Nama":                nama,
			"Tanggal_Lahir":       tanggalLahir,
			"Jenis_Kelamin":       jenisKelamin,
			"Tempat_Lahir":        tempatLahir,
			"NIK":                 nik,
			"Kelurahan":           kelurahan,
			"Kecamatan":           kecamatan,
			"Alamat":              alamat,
			"No_Telp":             noTelp,
			"Tanggal_Registrasi":  tanggalRegistrasi,
		})
	}
	return results, nil
}
