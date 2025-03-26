package services

import (
	"database/sql"
	"fmt"
	"strings"
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
func (s *PendaftaranService) RegisterPasienWithKunjungan(p models.Pasien, idPoli int, operatorID int, keluhanUtama string) (int64, int64, int64, string, int, string, error) {
    tx, err := s.DB.Begin()
    if err != nil {
        return 0, 0, 0, "", 0, "", err
    }

    // 1. Cek apakah NIK sudah ada di tabel Pasien
    var existingID int
    err = tx.QueryRow("SELECT id_pasien FROM Pasien WHERE NIK = ?", p.NIK).Scan(&existingID)
    if err == nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", fmt.Errorf("NIK sudah terdaftar")
    } else if err != sql.ErrNoRows {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", err
    }

    // 2. Insert data pasien ke tabel Pasien
    queryPasien := `
        INSERT INTO Pasien 
            (Nama, Tanggal_Lahir, Jenis_Kelamin, Tempat_Lahir, NIK, Kelurahan, Kecamatan, Alamat, No_Telp, kota_tinggal)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
        p.KotaTinggal,
    )
    if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", err
    }
    patientID, err := res.LastInsertId()
    if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", err
    }

    // 3. Generate id_rm dengan format "RM(Tahun)(5 digit)"
    tahun := time.Now().Year()
    var count int
    err = tx.QueryRow("SELECT count FROM Counter_RM WHERE tahun = ? FOR UPDATE", tahun).Scan(&count)
    if err == sql.ErrNoRows {
        _, err = tx.Exec("INSERT INTO Counter_RM (tahun, count) VALUES (?, 1)", tahun)
        if err != nil {
            tx.Rollback()
            return 0, 0, 0, "", 0, "", fmt.Errorf("failed to insert into Counter_RM: %v", err)
        }
        count = 1
    } else if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", fmt.Errorf("failed to select from Counter_RM: %v", err)
    } else {
        count++
        _, err = tx.Exec("UPDATE Counter_RM SET count = ? WHERE tahun = ?", count, tahun)
        if err != nil {
            tx.Rollback()
            return 0, 0, 0, "", 0, "", fmt.Errorf("failed to update Counter_RM: %v", err)
        }
    }
    formattedCount := fmt.Sprintf("%05d", count)
    idRM := fmt.Sprintf("RM%d%s", tahun, formattedCount)

    // 4. Insert ke Rekam_Medis dengan id_rm yang sudah dibuat
    queryRM := `INSERT INTO Rekam_Medis (id_rm, id_pasien) VALUES (?, ?)`
    _, err = tx.Exec(queryRM, idRM, patientID)
    if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", fmt.Errorf("failed to insert into Rekam_Medis: %v", err)
    }

    // 5. Buat record baru di Riwayat_Kunjungan (sementara catatan kosong)
    insertRK := `INSERT INTO Riwayat_Kunjungan (id_rm, Catatan) VALUES (?, ?)`
    res, err = tx.Exec(insertRK, idRM, "")
    if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", fmt.Errorf("failed to insert Riwayat_Kunjungan: %v", err)
    }
    idKunjungan, err := res.LastInsertId()
    if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get last insert id for Riwayat_Kunjungan: %v", err)
    }

    // 6. Hubungkan Riwayat_Kunjungan dengan Poliklinik melalui tabel Kunjungan_Poli
    insertKP := `INSERT INTO Kunjungan_Poli (id_poli, id_kunjungan) VALUES (?, ?)`
    _, err = tx.Exec(insertKP, idPoli, idKunjungan)
    if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", fmt.Errorf("failed to insert into Kunjungan_Poli: %v", err)
    }

    // 7. Hitung nomor antrian untuk poli yang dipilih pada hari ini (reset setiap hari)
    today := time.Now().Format("2006-01-02")
    var maxNomor sql.NullInt64
    err = tx.QueryRow("SELECT COALESCE(MAX(nomor_antrian), 0) FROM Antrian WHERE id_poli = ? AND DATE(created_at) = ?", idPoli, today).Scan(&maxNomor)
    if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get max nomor antrian: %v", err)
    }
    nomorAntrian := int64(1)
    if maxNomor.Valid && maxNomor.Int64 > 0 {
        nomorAntrian = maxNomor.Int64 + 1
    }

    // 8. Dapatkan id_status untuk "Menunggu" dari tabel Status_Antrian
    var idStatus int
    err = tx.QueryRow("SELECT id_status FROM Status_Antrian WHERE status = 'Menunggu' LIMIT 1").Scan(&idStatus)
    if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get id_status for 'Menunggu': %v", err)
    }

    // 9. Insert data antrian ke tabel Antrian, termasuk keluhan_utama
    insertAntrian := `
        INSERT INTO Antrian (id_pasien, id_poli, keluhan_utama, nomor_antrian, id_status, priority_order, created_at)
        VALUES (?, ?, ?, ?, ?, ?, NOW())
    `
    res, err = tx.Exec(insertAntrian, patientID, idPoli, keluhanUtama, nomorAntrian, idStatus, nomorAntrian)
    if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", fmt.Errorf("failed to insert into Antrian: %v", err)
    }
    idAntrian, err := res.LastInsertId()
    if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get id_antrian: %v", err)
    }

    // 10. Insert data billing ke tabel Billing
insertBilling := `
    INSERT INTO Billing (id_kunjungan, id_antrian, id_karyawan, id_billing_assessment, tipe_pembayaran, total, id_status, created_at, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, 1, NOW(), NOW())
`
res, err = tx.Exec(insertBilling, idKunjungan, idAntrian, nil, nil, nil, nil)
if err != nil {
    tx.Rollback()
    return 0, 0, 0, "", 0, "", fmt.Errorf("failed to insert into Billing: %v", err)
}
idBilling, err := res.LastInsertId()  // Simpan id billing yang baru dibuat
if err != nil {
    tx.Rollback()
    return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get id_billing: %v", err)
}

// 11. Update Riwayat_Kunjungan dengan id_antrian dan id_billing
updateRK := `
    UPDATE Riwayat_Kunjungan
    SET id_antrian = ?, id_billing = ?
    WHERE id_kunjungan = ?
`
_, err = tx.Exec(updateRK, idAntrian, idBilling, idKunjungan)
if err != nil {
    tx.Rollback()
    return 0, 0, 0, "", 0, "", fmt.Errorf("failed to update Riwayat_Kunjungan with id_antrian and id_billing: %v", err)
}


    // 12. Ambil nama_poli dari tabel Poliklinik
    var namaPoli string
    err = tx.QueryRow("SELECT nama_poli FROM Poliklinik WHERE id_poli = ?", idPoli).Scan(&namaPoli)
    if err != nil {
        tx.Rollback()
        return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get nama_poli: %v", err)
    }

    // Commit transaksi
    if err = tx.Commit(); err != nil {
        return 0, 0, 0, "", 0, "", err
    }

    return patientID, idAntrian, nomorAntrian, idRM, idStatus, namaPoli, nil
}



func (s *PendaftaranService) UpdatePasienAndRegisterKunjungan(p models.Pasien, idPoli int, keluhanUtama string) (int64, int64, int64, string, int, string, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, 0, 0, "", 0, "", err
	}

	// 1. Cari pasien berdasarkan NIK
	var idPasien int64
	err = tx.QueryRow("SELECT id_pasien FROM Pasien WHERE NIK = ?", p.NIK).Scan(&idPasien)
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("pasien with NIK %s not found: %v", p.NIK, err)
	}

	// 1a. Cek apakah pasien sudah mengambil antrian untuk poli ini hari ini
	today := time.Now().Format("2006-01-02")
	var lastAntrianPasien int64
	err = tx.QueryRow("SELECT id_pasien FROM Antrian WHERE id_poli = ? AND DATE(created_at) = ? ORDER BY created_at DESC LIMIT 1", idPoli, today).Scan(&lastAntrianPasien)
	if err == nil {
		if lastAntrianPasien == idPasien {
			tx.Rollback()
			return 0, 0, 0, "", 0, "", fmt.Errorf("duplicate entry: pasien dengan NIK %s baru saja mengambil antrian", p.NIK)
		}
	} else if err != sql.ErrNoRows {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("failed to check antrian duplicate: %v", err)
	}

	// 2. Update data pasien
	updateQuery := `
        UPDATE Pasien 
        SET Nama = ?, Tanggal_Lahir = ?, Jenis_Kelamin = ?, Tempat_Lahir = ?, Kelurahan = ?, Kecamatan = ?, kota_tinggal = ?, Alamat = ?, No_Telp = ?
        WHERE id_pasien = ?
    `
	_, err = tx.Exec(updateQuery,
		p.Nama,
		p.TanggalLahir,
		p.JenisKelamin,
		p.TempatLahir,
		p.Kelurahan,
		p.Kecamatan,
		p.KotaTinggal,
		p.Alamat,
		p.NoTelp,
		idPasien,
	)
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("failed to update pasien: %v", err)
	}

	// 3. Ambil ID_RM dari Rekam_Medis
	var idRM string
	err = tx.QueryRow("SELECT id_rm FROM Rekam_Medis WHERE id_pasien = ? ORDER BY created_at DESC LIMIT 1", idPasien).Scan(&idRM)
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get Rekam_Medis for pasien: %v", err)
	}

	// 4. Insert record baru di Riwayat_Kunjungan untuk kunjungan tambahan
	insertRK := `
        INSERT INTO Riwayat_Kunjungan (id_rm, Catatan)
        VALUES (?, ?)
    `
	res, err := tx.Exec(insertRK, idRM, "")
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("failed to insert Riwayat_Kunjungan: %v", err)
	}
	idKunjungan, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get last insert id for Riwayat_Kunjungan: %v", err)
	}

	// 5. Hubungkan Riwayat_Kunjungan dengan Poliklinik melalui tabel Kunjungan_Poli
	insertKP := `INSERT INTO Kunjungan_Poli (id_poli, id_kunjungan) VALUES (?, ?)`
	_, err = tx.Exec(insertKP, idPoli, idKunjungan)
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("failed to insert into Kunjungan_Poli: %v", err)
	}

	// 6. Hitung nomor antrian untuk poli yang dipilih hari ini
	var maxNomor sql.NullInt64
	err = tx.QueryRow("SELECT COALESCE(MAX(nomor_antrian), 0) FROM Antrian WHERE id_poli = ? AND DATE(created_at) = ?", idPoli, today).Scan(&maxNomor)
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get max nomor antrian: %v", err)
	}
	nomorAntrian := int64(1)
	if maxNomor.Valid && maxNomor.Int64 > 0 {
		nomorAntrian = maxNomor.Int64 + 1
	}

	// 7. Dapatkan id_status untuk "Menunggu"
	var idStatus int
	err = tx.QueryRow("SELECT id_status FROM Status_Antrian WHERE status = 'Menunggu' LIMIT 1").Scan(&idStatus)
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get id_status for 'Menunggu': %v", err)
	}

	// 8. Insert data antrian ke tabel Antrian
	insertAntrian := `
        INSERT INTO Antrian (id_pasien, id_poli, keluhan_utama, nomor_antrian, id_status, priority_order, created_at)
        VALUES (?, ?, ?, ?, ?, ?, NOW())
    `
	res, err = tx.Exec(insertAntrian, idPasien, idPoli, keluhanUtama, nomorAntrian, idStatus, nomorAntrian)
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("failed to insert into Antrian: %v", err)
	}
	idAntrian, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get id_antrian: %v", err)
	}

	// 9. Insert data billing ke tabel Billing
insertBilling := `
    INSERT INTO Billing (id_kunjungan, id_antrian, id_karyawan, id_billing_assessment, tipe_pembayaran, total, id_status, created_at, updated_at)
    VALUES (?, ?, NULL, NULL, NULL, NULL, 1, NOW(), NOW())
`
res, err = tx.Exec(insertBilling, idKunjungan, idAntrian)
if err != nil {
    tx.Rollback()
    return 0, 0, 0, "", 0, "", fmt.Errorf("failed to insert into Billing: %v", err)
}
idBilling, err := res.LastInsertId() // Tangkap id_billing yang baru dibuat
if err != nil {
    tx.Rollback()
    return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get id_billing: %v", err)
}

// 10. Update Riwayat_Kunjungan dengan id_antrian dan id_billing
updateRK := `
    UPDATE Riwayat_Kunjungan
    SET id_antrian = ?, id_billing = ?
    WHERE id_kunjungan = ?
`
_, err = tx.Exec(updateRK, idAntrian, idBilling, idKunjungan)
if err != nil {
    tx.Rollback()
    return 0, 0, 0, "", 0, "", fmt.Errorf("failed to update Riwayat_Kunjungan with id_antrian and id_billing: %v", err)
}


	// 11. Ambil nama_poli dari tabel Poliklinik
	var namaPoli string
	err = tx.QueryRow("SELECT nama_poli FROM Poliklinik WHERE id_poli = ?", idPoli).Scan(&namaPoli)
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, "", 0, "", fmt.Errorf("failed to get nama_poli: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return 0, 0, 0, "", 0, "", err
	}

	return idPasien, idAntrian, nomorAntrian, idRM, idStatus, namaPoli, nil
}




func (s *PendaftaranService) GetAllPasienDataFiltered(namaFilter string, page, limit int) ([]map[string]interface{}, error) {
	// Base query untuk mengambil data pasien
	query := `
		SELECT id_pasien, nama, tanggal_lahir, jenis_kelamin, tempat_lahir, nik, kelurahan, kecamatan, kota_tinggal, alamat, no_telp, tanggal_regist
		FROM Pasien
	`
	conditions := []string{}
	args := []interface{}{}

	// Jika ada filter nama, tambahkan kondisi WHERE
	if strings.TrimSpace(namaFilter) != "" {
		conditions = append(conditions, "nama LIKE ?")
		args = append(args, "%"+namaFilter+"%")
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY tanggal_regist DESC"

	// Hitung offset berdasarkan page dan limit
	offset := (page - 1) * limit
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var idPasien int
		var nama string
		var tanggalLahir string
		var jenisKelamin, tempatLahir, nik, kelurahan, kecamatan, kotaTinggal, alamat, noTelp string
		var tanggalRegist string

		if err := rows.Scan(&idPasien, &nama, &tanggalLahir, &jenisKelamin, &tempatLahir, &nik, &kelurahan, &kecamatan, &kotaTinggal, &alamat, &noTelp, &tanggalRegist); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}

		record := map[string]interface{}{
			"ID_Pasien":      idPasien,
			"Nama":           nama,
			"Tanggal_Lahir":  tanggalLahir,
			"Jenis_Kelamin":  jenisKelamin,
			"Tempat_Lahir":   tempatLahir,
			"NIK":            nik,
			"Kelurahan":      kelurahan,
			"Kecamatan":      kecamatan,
			"Kota_Tinggal":   kotaTinggal,
			"Alamat":         alamat,
			"No_Telp":        noTelp,
			"Tanggal_Regist": tanggalRegist,
		}
		results = append(results, record)
	}
	return results, nil
}


func (s *PendaftaranService) TundaPasien(idAntrian int) error {
    // 1. Periksa apakah antrian ada
    var exists bool
    err := s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM Antrian WHERE id_antrian = ?)", idAntrian).Scan(&exists)
    if err != nil {
        return fmt.Errorf("gagal memeriksa keberadaan antrian: %v", err)
    }
    if !exists {
        return fmt.Errorf("antrian dengan id %d tidak ditemukan", idAntrian)
    }

    // 2. Dapatkan id_status untuk "Ditunda"
    var idStatus int
    err = s.DB.QueryRow("SELECT id_status FROM Status_Antrian WHERE status = 'Ditunda' LIMIT 1").Scan(&idStatus)
    if err != nil {
        if err == sql.ErrNoRows {
            return fmt.Errorf("status 'Ditunda' tidak ditemukan di tabel Status_Antrian")
        }
        return fmt.Errorf("gagal mendapatkan id_status untuk 'Ditunda': %v", err)
    }

    // 3. Update status antrian
    query := `UPDATE Antrian SET id_status = ? WHERE id_antrian = ?`
    result, err := s.DB.Exec(query, idStatus, idAntrian)
    if err != nil {
        return fmt.Errorf("gagal mengupdate status antrian: %v", err)
    }

    // 4. Periksa apakah ada baris yang terupdate
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("gagal memeriksa jumlah baris yang terupdate: %v", err)
    }
    if rowsAffected == 0 {
        return fmt.Errorf("tidak ada baris yang terupdate, antrian dengan id %d mungkin tidak ada", idAntrian)
    }

    return nil
}


func (s *PendaftaranService) RescheduleAntrianPriority(idAntrian int) (int64, error) {
    // 1. Periksa apakah antrian ada dan dalam status "Ditunda", sekaligus ambil id_poli
    var currentStatus int
    var idPoli int
    err := s.DB.QueryRow("SELECT id_status, id_poli FROM Antrian WHERE id_antrian = ?", idAntrian).Scan(&currentStatus, &idPoli)
    if err != nil {
        if err == sql.ErrNoRows {
            return 0, fmt.Errorf("antrian dengan id %d tidak ditemukan", idAntrian)
        }
        return 0, fmt.Errorf("gagal menemukan antrian: %v", err)
    }
    if currentStatus != 2 { // Misalkan 2 adalah id_status untuk "Ditunda"
        return 0, fmt.Errorf("antrian tidak dalam status 'Ditunda', status saat ini: %d", currentStatus)
    }

    // 2. Tentukan hari ini
    today := time.Now().Format("2006-01-02")

    // 3. Cari MIN(nomor_antrian) dari antrian dengan status "Menunggu" (id_status = 1)
    var minWaiting sql.NullInt64
    queryMin := `
        SELECT MIN(nomor_antrian)
        FROM Antrian
        WHERE id_poli = ? AND DATE(created_at) = ? AND id_status = 1
    `
    err = s.DB.QueryRow(queryMin, idPoli, today).Scan(&minWaiting)
    if err != nil {
        return 0, fmt.Errorf("gagal mendapatkan nomor antrian minimum untuk 'Menunggu': %v", err)
    }

    // 4. Hitung jumlah antrian waiting (id_status = 1) untuk id_poli hari ini
    var countWaiting int
    queryCount := `
        SELECT COUNT(*)
        FROM Antrian
        WHERE id_poli = ? AND DATE(created_at) = ? AND id_status = 1
    `
    err = s.DB.QueryRow(queryCount, idPoli, today).Scan(&countWaiting)
    if err != nil {
        return 0, fmt.Errorf("gagal menghitung jumlah antrian menunggu: %v", err)
    }

    // 5. Tentukan newPriority
    var newPriority int64
    if minWaiting.Valid {
        if countWaiting >= 2 {
            newPriority = minWaiting.Int64 + 2
        } else {
            newPriority = minWaiting.Int64 + 1
        }
    } else {
        newPriority = 1 // Jika tidak ada antrian menunggu
    }

    // 6. Ambil id_status untuk "Menunggu"
    var waitingStatus int
    err = s.DB.QueryRow("SELECT id_status FROM Status_Antrian WHERE status = 'Menunggu' LIMIT 1").Scan(&waitingStatus)
    if err != nil {
        if err == sql.ErrNoRows {
            return 0, fmt.Errorf("status 'Menunggu' tidak ditemukan di tabel Status_Antrian")
        }
        return 0, fmt.Errorf("gagal mendapatkan id_status untuk 'Menunggu': %v", err)
    }

    // 7. Update record antrian
    updateQuery := `
        UPDATE Antrian
        SET priority_order = ?, id_status = ?
        WHERE id_antrian = ?
    `
    result, err := s.DB.Exec(updateQuery, newPriority, waitingStatus, idAntrian)
    if err != nil {
        return 0, fmt.Errorf("gagal mengupdate antrian: %v", err)
    }

    // 8. Periksa apakah ada baris yang terupdate
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return 0, fmt.Errorf("gagal memeriksa jumlah baris yang terupdate: %v", err)
    }
    if rowsAffected == 0 {
        return 0, fmt.Errorf("tidak ada baris yang terupdate, antrian dengan id %d mungkin tidak ada", idAntrian)
    }

    return newPriority, nil
}

// GetAntrianToday mengambil data antrian hari ini dengan join ke Pasien, Rekam_Medis, Poliklinik, dan Status_Antrian.
// Jika statusFilter tidak kosong, query akan memfilter berdasarkan status.
func (s *PendaftaranService) GetAntrianToday(statusFilter string) ([]map[string]interface{}, error) {
    query := `
        SELECT 
            p.id_pasien,
            p.nama,
            rm.id_rm,
            a.id_poli,
            pol.nama_poli,
            a.nomor_antrian,
            a.id_antrian,
            a.id_status,
            sa.status,
            a.priority_order
        FROM Antrian a
        JOIN Pasien p ON a.id_pasien = p.id_pasien
        JOIN Rekam_Medis rm ON p.id_pasien = rm.id_pasien
        JOIN Poliklinik pol ON a.id_poli = pol.id_poli
        JOIN Status_Antrian sa ON a.id_status = sa.id_status
        WHERE DATE(a.created_at) = CURDATE()
    `
    // Jika statusFilter disediakan, tambahkan filter.
    params := []interface{}{}
    if statusFilter != "" {
        query += " AND sa.status = ?"
        params = append(params, statusFilter)
    }
    // Urutkan berdasarkan nomor antrian
    query += " ORDER BY a.nomor_antrian"

    rows, err := s.DB.Query(query, params...)
    if err != nil {
        return nil, fmt.Errorf("query error: %v", err)
    }
    defer rows.Close()

    var list []map[string]interface{}
    for rows.Next() {
        var idPasien int
        var nama string
        var idRM sql.NullString
        var idPoli int
        var namaPoli sql.NullString
        var nomorAntrian int
        var idAntrian int 
        var idStatus int
        var status sql.NullString
        var priorityOrder sql.NullInt64

        // Sesuaikan urutan scan dengan urutan kolom di SELECT
        if err := rows.Scan(&idPasien, &nama, &idRM, &idPoli, &namaPoli, &nomorAntrian, &idAntrian, &idStatus, &status, &priorityOrder); err != nil {
            return nil, fmt.Errorf("scan error: %v", err)
        }

        record := map[string]interface{}{
            "id_pasien":     idPasien,
            "nama":          nama,
            "id_rm":         nil,
            "id_poli":       idPoli,
            "nama_poli":     nil,
            "nomor_antrian": nomorAntrian,
            "id_antrian":    idAntrian,
            "id_status":     idStatus,
            "status":        nil,
            "priority_order": nil,
        }
        if idRM.Valid {
            record["id_rm"] = idRM.String
        }
        if namaPoli.Valid {
            record["nama_poli"] = namaPoli.String
        }
        if status.Valid {
            record["status"] = status.String
        }
        if priorityOrder.Valid {
            record["priority_order"] = priorityOrder.Int64
        }
        list = append(list, record)
    }
    return list, nil
}

func (s *PendaftaranService) GetAllStatusAntrian() ([]map[string]interface{}, error) {
	query := "SELECT id_status, status FROM Status_Antrian"
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var idStatus int
		var status string
		if err := rows.Scan(&idStatus, &status); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}
		record := map[string]interface{}{
			"id_status": idStatus,
			"status":    status,
		}
		list = append(list, record)
	}
	return list, nil
}

func (s *PendaftaranService) BatalkanAntrian(idAntrian int) error {
    // 1. Update status antrian (contoh: update ke status batal, misalnya 3)
    updateAntrianQuery := "UPDATE Antrian SET id_status = ? WHERE id_antrian = ?"
    result, err := s.DB.Exec(updateAntrianQuery, 3, idAntrian)
    if err != nil {
        return fmt.Errorf("gagal membatalkan antrian: %v", err)
    }
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("gagal memeriksa update antrian: %v", err)
    }
    if rowsAffected == 0 {
        return fmt.Errorf("antrian dengan id %d tidak ditemukan", idAntrian)
    }

    // 2. Update id_status billing menjadi 4 untuk antrian ini.
    // Query ini mengupdate tabel Billing dengan join ke Riwayat_Kunjungan berdasarkan id_kunjungan.
    updateBillingQuery := `
        UPDATE Billing b
        JOIN Riwayat_Kunjungan rk ON b.id_kunjungan = rk.id_kunjungan
        SET b.id_status = 4
        WHERE rk.id_antrian = ?
    `
    result, err = s.DB.Exec(updateBillingQuery, idAntrian)
    if err != nil {
        return fmt.Errorf("gagal mengupdate status billing: %v", err)
    }
    // Opsional: periksa apakah ada baris yang terupdate
    rowsAffected, err = result.RowsAffected()
    if err != nil {
        return fmt.Errorf("gagal memeriksa update status billing: %v", err)
    }
    if rowsAffected == 0 {
        // Jika tidak ada baris yang diupdate, Anda bisa menganggapnya sebagai kondisi valid atau error,
        // tergantung pada kebutuhan aplikasi.
    }

    return nil
}
