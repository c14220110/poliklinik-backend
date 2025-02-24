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
// Catatan: Kolom Status di tabel Antrian telah diganti menjadi id_status yang mengacu ke tabel Status_Antrian.
func (s *PendaftaranService) RegisterPasienWithKunjungan(p models.Pasien, idPoli int, operatorID int) (int64, int64, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, 0, err
	}

	// 1. Cek apakah NIK sudah ada di tabel Pasien
	var existingID int
	err = tx.QueryRow("SELECT id_pasien FROM Pasien WHERE NIK = ?", p.NIK).Scan(&existingID)
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
		INSERT INTO Rekam_Medis (id_pasien)
		VALUES (?)
	`
	_, err = tx.Exec(queryRM, patientID)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 4. Pencatatan Riwayat Kunjungan (insert hanya ID_RM)
	var idRM int64
	err = tx.QueryRow("SELECT id_rm FROM Rekam_Medis WHERE id_pasien = ? ORDER BY created_at DESC LIMIT 1", patientID).Scan(&idRM)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}
	queryRK := `
		INSERT INTO Riwayat_Kunjungan (id_rm)
		VALUES (?)
	`
	_, err = tx.Exec(queryRK, idRM)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 5. Hubungan Kunjungan dengan Poli: Insert ke tabel Kunjungan_Poli
	queryKP := `
		INSERT INTO Kunjungan_Poli (id_poli, id_kunjungan)
		VALUES (?, (SELECT id_kunjungan FROM Riwayat_Kunjungan WHERE id_rm = ? ORDER BY created_at DESC LIMIT 1))
	`
	_, err = tx.Exec(queryKP, idPoli, idRM)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}

	// 6. Pembuatan Nomor Antrian: Hitung nomor antrian untuk poli yang dipilih pada hari ini (reset tiap hari)
	today := time.Now().Format("2006-01-02")
	var maxNomor sql.NullInt64
	queryMax := `
		SELECT COALESCE(MAX(nomor_antrian), 0)
		FROM Antrian
		WHERE id_poli = ? AND DATE(created_at) = ?
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

	// 7. Dapatkan id_status untuk "Menunggu" dari tabel Status_Antrian
	var idStatus int
	err = tx.QueryRow("SELECT id_status FROM Status_Antrian WHERE status = 'Menunggu' LIMIT 1").Scan(&idStatus)
	if err != nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("failed to get id_status for 'Menunggu': %v", err)
	}

	// 8. Insert data antrian ke tabel Antrian dengan id_status dari tabel helper
	// Pastikan priority_order diisi dengan nilai yang sama dengan nomor_antrian
	queryAntrian := `
		INSERT INTO Antrian (id_pasien, id_poli, nomor_antrian, id_status, priority_order, created_at)
		VALUES (?, ?, ?, ?, ?, NOW())
	`
	_, err = tx.Exec(queryAntrian, patientID, idPoli, nomorAntrian, idStatus, nomorAntrian)
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
func (s *PendaftaranService) UpdatePasienAndRegisterKunjungan(p models.Pasien, idPoli int) (int64, int64, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, 0, err
	}

	// 1. Cari pasien berdasarkan NIK (sebagai kunci pencarian)
	var idPasien int64
	err = tx.QueryRow("SELECT id_pasien FROM Pasien WHERE NIK = ?", p.NIK).Scan(&idPasien)
	if err != nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("pasien with NIK %s not found: %v", p.NIK, err)
	}

	// 1a. Cek record antrian terbaru untuk poli hari ini, apakah sudah ada record dengan id_pasien yang sama.
	today := time.Now().Format("2006-01-02")
	var lastAntrianPasien int64
	err = tx.QueryRow("SELECT id_pasien FROM Antrian WHERE id_poli = ? AND DATE(created_at) = ? ORDER BY created_at DESC LIMIT 1", idPoli, today).Scan(&lastAntrianPasien)
	if err == nil {
		if lastAntrianPasien == idPasien {
			tx.Rollback()
			return 0, 0, fmt.Errorf("duplicate entry: pasien dengan NIK %s baru saja mengambil antrian, ", p.NIK)
		}
	} else if err != sql.ErrNoRows {
		tx.Rollback()
		return 0, 0, fmt.Errorf("failed to check antrian duplicate: %v", err)
	}
	// Jika tidak ada record (sql.ErrNoRows), lanjutkan.

	// 2. Update data pasien (tidak mengubah NIK)
	updateQuery := `
		UPDATE Pasien 
		SET Nama = ?, Tanggal_Lahir = ?, Jenis_Kelamin = ?, Tempat_Lahir = ?, Kelurahan = ?, Kecamatan = ?, Alamat = ?, No_Telp = ?
		WHERE id_pasien = ?
	`
	_, err = tx.Exec(updateQuery,
		p.Nama,
		p.TanggalLahir,
		p.JenisKelamin,
		p.TempatLahir,
		p.Kelurahan,
		p.Kecamatan,
		p.Alamat,
		p.NoTelp,
		idPasien,
	)
	if err != nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("failed to update pasien: %v", err)
	}

	// 3. Ambil ID_RM (Rekam_Medis) untuk pasien tersebut
	var idRM int64
	err = tx.QueryRow("SELECT id_rm FROM Rekam_Medis WHERE id_pasien = ? ORDER BY created_at DESC LIMIT 1", idPasien).Scan(&idRM)
	if err != nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("failed to get Rekam_Medis for pasien: %v", err)
	}

	// 4. Buat record baru di Riwayat_Kunjungan untuk kunjungan tambahan.
	insertRK := `
		INSERT INTO Riwayat_Kunjungan (id_rm, Catatan)
		VALUES (?, ?)
	`
	res, err := tx.Exec(insertRK, idRM, "")
	if err != nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("failed to insert Riwayat_Kunjungan: %v", err)
	}
	idKunjungan, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("failed to get last insert id for Riwayat_Kunjungan: %v", err)
	}

	// 5. Hubungkan Riwayat_Kunjungan dengan Poliklinik melalui tabel Kunjungan_Poli.
	insertKP := `INSERT INTO Kunjungan_Poli (id_poli, id_kunjungan) VALUES (?, ?)`
	_, err = tx.Exec(insertKP, idPoli, idKunjungan)
	if err != nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("failed to insert into Kunjungan_Poli: %v", err)
	}

	// 6. Hitung nomor antrian untuk poli yang dipilih pada hari ini (reset setiap hari).
	var maxNomor sql.NullInt64
	err = tx.QueryRow("SELECT COALESCE(MAX(nomor_antrian), 0) FROM Antrian WHERE id_poli = ? AND DATE(created_at) = ?", idPoli, today).Scan(&maxNomor)
	if err != nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("failed to get max nomor antrian: %v", err)
	}
	nomorAntrian := int64(1)
	if maxNomor.Valid && maxNomor.Int64 > 0 {
		nomorAntrian = maxNomor.Int64 + 1
	}

	// 7. Dapatkan id_status untuk "Menunggu" dari tabel Status_Antrian.
	var idStatus int
	err = tx.QueryRow("SELECT id_status FROM Status_Antrian WHERE status = 'Menunggu' LIMIT 1").Scan(&idStatus)
	if err != nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("failed to get id_status for 'Menunggu': %v", err)
	}

	// 8. Insert data antrian ke tabel Antrian dengan id_status dan priority_order diisi dengan nomor_antrian.
	insertAntrian := `
		INSERT INTO Antrian (id_pasien, id_poli, nomor_antrian, id_status, priority_order, created_at)
		VALUES (?, ?, ?, ?, ?, NOW())
	`
	_, err = tx.Exec(insertAntrian, idPasien, idPoli, nomorAntrian, idStatus, nomorAntrian)
	if err != nil {
		tx.Rollback()
		return 0, 0, fmt.Errorf("failed to insert into Antrian: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}

	return idKunjungan, nomorAntrian, nil
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

func (s *PendaftaranService) TundaPasien(idAntrian int) error {
	// Dapatkan id_status untuk "Ditunda"
	var idStatus int
	err := s.DB.QueryRow("SELECT id_status FROM Status_Antrian WHERE status = 'Ditunda' LIMIT 1").Scan(&idStatus)
	if err != nil {
		return fmt.Errorf("failed to get id_status for 'Ditunda': %v", err)
	}

	query := `UPDATE Antrian SET id_status = ? WHERE id_antrian = ?`
	_, err = s.DB.Exec(query, idStatus, idAntrian)
	return err
}


// RescheduleAntrianPriority mengupdate baris antrian (yang statusnya Ditunda)
// dengan mengubah priority_order dan id_status (ke "Menunggu") tanpa mengubah nomor_antrian.
// Logika:
// 1. Pastikan record id_antrian ada di id_poli dan statusnya adalah Ditunda (misal: 1).
// 2. Cari MIN(nomor_antrian) di antara record waiting (id_status = 0) untuk id_poli hari ini.
// 3. Hitung newPriority = MIN_waiting + 2 jika count waiting >= 2, atau +1 jika count waiting kurang dari 2.
// 4. Update record antrian tersebut dengan newPriority dan ubah id_status menjadi nilai untuk "Menunggu".
func (s *PendaftaranService) RescheduleAntrianPriority(idAntrian int, idPoli int) (int64, error) {
	// 1. Periksa apakah record id_antrian ada di id_poli dengan status Ditunda (misalnya id_status = 1)
	var currentStatus int
	err := s.DB.QueryRow("SELECT id_status FROM Antrian WHERE id_antrian = ? AND id_poli = ?", idAntrian, idPoli).Scan(&currentStatus)
	if err != nil {
		return 0, fmt.Errorf("failed to find antrian: %v", err)
	}
	if currentStatus != 1 {
		return 0, fmt.Errorf("antrian is not in 'Ditunda' status")
	}

	// 2. Tentukan hari ini
	today := time.Now().Format("2006-01-02")

	// 3. Cari MIN(nomor_antrian) dari antrian dengan status "Menunggu" (id_status = 0)
	var minWaiting sql.NullInt64
	queryMin := `
		SELECT MIN(nomor_antrian)
		FROM Antrian
		WHERE id_poli = ? AND DATE(created_at) = ? AND id_status = 0
	`
	err = s.DB.QueryRow(queryMin, idPoli, today).Scan(&minWaiting)
	if err != nil {
		return 0, fmt.Errorf("failed to get minimum waiting nomor_antrian: %v", err)
	}

	// 4. Hitung jumlah antrian waiting untuk id_poli hari ini
	var countWaiting int
	queryCount := `
		SELECT COUNT(*)
		FROM Antrian
		WHERE id_poli = ? AND DATE(created_at) = ? AND id_status = 0
	`
	err = s.DB.QueryRow(queryCount, idPoli, today).Scan(&countWaiting)
	if err != nil {
		return 0, fmt.Errorf("failed to count waiting antrian: %v", err)
	}

	// 5. Tentukan newPriority:
	var newPriority int64
	if minWaiting.Valid {
		if countWaiting >= 2 {
			newPriority = minWaiting.Int64 + 2
		} else {
			newPriority = minWaiting.Int64 + 1
		}
	} else {
		// Jika tidak ada record waiting, misalnya newPriority = 1
		newPriority = 1
	}

	// 6. Ambil id_status untuk "Menunggu" dari tabel Status_Antrian
	var waitingStatus int
	err = s.DB.QueryRow("SELECT id_status FROM Status_Antrian WHERE status = 'Menunggu' LIMIT 1").Scan(&waitingStatus)
	if err != nil {
		return 0, fmt.Errorf("failed to get id_status for 'Menunggu': %v", err)
	}

	// 7. Update record antrian: jangan ubah nomor_antrian, hanya update priority_order dan id_status
	updateQuery := `
		UPDATE Antrian
		SET priority_order = ?, id_status = ?
		WHERE id_antrian = ?
	`
	_, err = s.DB.Exec(updateQuery, newPriority, waitingStatus, idAntrian)
	if err != nil {
		return 0, fmt.Errorf("failed to update antrian: %v", err)
	}
	return newPriority, nil
}