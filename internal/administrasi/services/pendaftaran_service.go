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

// RegisterPasienWithKunjungan mendaftarkan pasien, RM, kunjungan, antrian, dan billing.
func (s *PendaftaranService) RegisterPasienWithKunjungan(
	p models.Pasien,
	idPoli, operatorID int,
	keluhanUtama, namaPenanggungJawab string,
) (patientID int64, idAntrian int64, nomorAntrian int64, idRM string, idStatus int,
	namaPoli string, err error) {

	// ---------- MULAI TRANSAKSI ----------
	tx, err := s.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// 1. Pastikan NIK belum terdaftar
	var existingID int
	if err = tx.QueryRow(`SELECT id_pasien FROM Pasien WHERE NIK = ?`, p.NIK).Scan(&existingID); err == nil {
		err = fmt.Errorf("NIK sudah terdaftar")
		return
	} else if err != sql.ErrNoRows {
		return
	}

	// 2. Masukkan data pasien
	res, err := tx.Exec(`
		INSERT INTO Pasien
		  (Nama, Tanggal_Lahir, Jenis_Kelamin, Tempat_Lahir,
		   NIK, Kelurahan, Kecamatan, Alamat, No_Telp, kota_tinggal,
		   id_agama, status_perkawinan, pekerjaan)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Nama, p.TanggalLahir, p.JenisKelamin, p.TempatLahir,
		p.NIK, p.Kelurahan, p.Kecamatan, p.Alamat, p.NoTelp, p.KotaTinggal,
		p.IDAgama, p.StatusPerkawinan, p.Pekerjaan,
	)
	if err != nil {
		return
	}
	if patientID, err = res.LastInsertId(); err != nil {
		return
	}

	// 3. Buat id_rm (Counter_RM)
	tahun := time.Now().Year()
	var count int
	switch err = tx.QueryRow(`SELECT count FROM Counter_RM WHERE tahun = ? FOR UPDATE`, tahun).Scan(&count); {
	case err == sql.ErrNoRows:
		if _, err = tx.Exec(`INSERT INTO Counter_RM (tahun, count) VALUES (?, 1)`, tahun); err != nil {
			err = fmt.Errorf("failed insert Counter_RM: %v", err)
			return
		}
		count = 1
	case err == nil:
		count++
		if _, err = tx.Exec(`UPDATE Counter_RM SET count = ? WHERE tahun = ?`, count, tahun); err != nil {
			err = fmt.Errorf("failed update Counter_RM: %v", err)
			return
		}
	default:
		err = fmt.Errorf("failed select Counter_RM: %v", err)
		return
	}
	idRM = fmt.Sprintf("RM%d%05d", tahun, count)

	// 4. Rekam Medis
	if _, err = tx.Exec(`INSERT INTO Rekam_Medis (id_rm, id_pasien) VALUES (?, ?)`, idRM, patientID); err != nil {
		err = fmt.Errorf("insert Rekam_Medis: %v", err)
		return
	}

	// 5. Riwayat_Kunjungan
	if res, err = tx.Exec(`INSERT INTO Riwayat_Kunjungan (id_rm, catatan) VALUES (?, '')`, idRM); err != nil {
		err = fmt.Errorf("insert Riwayat_Kunjungan: %v", err)
		return
	}
	var idKunjungan int64
	if idKunjungan, err = res.LastInsertId(); err != nil {
		err = fmt.Errorf("lastInsertId Riwayat_Kunjungan: %v", err)
		return
	}

	// 6. Kunjungan_Poli
	if _, err = tx.Exec(`INSERT INTO Kunjungan_Poli (id_poli, id_kunjungan) VALUES (?, ?)`, idPoli, idKunjungan); err != nil {
		err = fmt.Errorf("insert Kunjungan_Poli: %v", err)
		return
	}

	// 7. Nomor antrian hari ini
	today := time.Now().Format("2006-01-02")
	var maxNomor sql.NullInt64
	if err = tx.QueryRow(
		`SELECT COALESCE(MAX(nomor_antrian),0) FROM Antrian WHERE id_poli = ? AND DATE(created_at) = ?`,
		idPoli, today,
	).Scan(&maxNomor); err != nil {
		err = fmt.Errorf("max nomor_antrian: %v", err)
		return
	}
	nomorAntrian = 1
	if maxNomor.Valid && maxNomor.Int64 > 0 {
		nomorAntrian = maxNomor.Int64 + 1
	}

	// 8. id_status “Menunggu”
	if err = tx.QueryRow(`SELECT id_status FROM Status_Antrian WHERE status = 'Menunggu' LIMIT 1`).Scan(&idStatus); err != nil {
		err = fmt.Errorf("id_status Menunggu: %v", err)
		return
	}

	// 9. Antrian
	if res, err = tx.Exec(`
		INSERT INTO Antrian
		  (id_pasien, id_poli, keluhan_utama, nomor_antrian,
		   id_status, priority_order, created_at, nama_penanggung_jawab)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), ?)`,
		patientID, idPoli, keluhanUtama, nomorAntrian,
		idStatus, nomorAntrian, namaPenanggungJawab,
	); err != nil {
		err = fmt.Errorf("insert Antrian: %v", err)
		return
	}
	if idAntrian, err = res.LastInsertId(); err != nil {
		err = fmt.Errorf("lastInsertId Antrian: %v", err)
		return
	}

	// 10. Billing  **SUDAH DISESUAIKAN DENGAN id_assessment**
	if _, err = tx.Exec(`
		INSERT INTO Billing
		  (id_kunjungan, id_antrian, id_karyawan, id_assessment,
		   tipe_pembayaran, total, id_status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, NOW(), NOW())`,
		idKunjungan, idAntrian, operatorID,
		nil,      // id_assessment  (belum ada / NULL)
		nil, nil, // tipe_pembayaran, total
	); err != nil {
		err = fmt.Errorf("insert Billing: %v", err)
		return
	}

	// 11. Update Riwayat_Kunjungan ← id_antrian
	if _, err = tx.Exec(`UPDATE Riwayat_Kunjungan SET id_antrian = ? WHERE id_kunjungan = ?`,
		idAntrian, idKunjungan); err != nil {
		err = fmt.Errorf("update Riwayat_Kunjungan: %v", err)
		return
	}

	// 12. Ambil nama poli
	if err = tx.QueryRow(`SELECT nama_poli FROM Poliklinik WHERE id_poli = ?`, idPoli).Scan(&namaPoli); err != nil {
		err = fmt.Errorf("select nama_poli: %v", err)
		return
	}

	// ---------- COMMIT ----------
	err = tx.Commit()
	return
}



// UpdatePasienAndRegisterKunjungan update pasien & buat antrian baru tanpa id_billing di RK
// UpdatePasienAndRegisterKunjungan updates pasien & creates a new antrian without id_billing in RK
func (s *PendaftaranService) UpdatePasienAndRegisterKunjungan(
	p models.Pasien,
	idPoli int,
	keluhanUtama string,
	namaPenanggungJawab string,
) (idPasien, idAntrian, nomorAntrian int64, idRM string, idStatus int, namaPoli string, err error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	// 1. Cari pasien by NIK
	err = tx.QueryRow("SELECT id_pasien FROM Pasien WHERE NIK = ?", p.NIK).Scan(&idPasien)
	if err != nil {
		err = fmt.Errorf("pasien with NIK %s not found: %v", p.NIK, err)
		return
	}

	// 1a. Cek duplicate antrian hari ini
	today := time.Now().Format("2006-01-02")
	var lastAntrianPasien int64
	err = tx.QueryRow(`
		SELECT id_pasien 
		FROM Antrian 
		WHERE id_poli = ? AND DATE(created_at)=? 
		ORDER BY created_at DESC 
		LIMIT 1`,
		idPoli, today,
	).Scan(&lastAntrianPasien)
	if err == nil && lastAntrianPasien == idPasien {
		err = fmt.Errorf("duplicateili entry: pasien dengan NIK %s baru saja mengambil antrian", p.NIK)
		return
	} else if err != nil && err != sql.ErrNoRows {
		err = fmt.Errorf("failed to check antrian duplicate: %v", err)
		return
	}

	// 2. Update data Pasien
	_, err = tx.Exec(`
		UPDATE Pasien 
		SET Nama=?, Tanggal_Lahir=?, Jenis_Kelamin=?, Tempat_Lahir=?,
		    Kelurahan=?, Kecamatan=?, kota_tinggal=?, Alamat=?, No_Telp=?,
		    id_agama=?, status_perkawinan=?, pekerjaan=?
		WHERE id_pasien = ?`,
		p.Nama, p.TanggalLahir, p.JenisKelamin, p.TempatLahir,
		p.Kelurahan, p.Kecamatan, p.KotaTinggal, p.Alamat, p.NoTelp,
		p.IDAgama, p.StatusPerkawinan, p.Pekerjaan,
		idPasien,
	)
	if err != nil {
		err = fmt.Errorf("failed to update pasien: %v", err)
		return
	}

	// 3. Ambil id_rm terbaru
	err = tx.QueryRow(`
		SELECT id_rm 
		FROM Rekam_Medis 
		WHERE id_pasien=? 
		ORDER BY created_at DESC 
		LIMIT 1`,
		idPasien,
	).Scan(&idRM)
	if err != nil {
		err = fmt.Errorf("failed to get Rekam_Medis for pasien: %v", err)
		return
	}

	// 4. Insert Riwayat_Kunjungan
	var idKunjungan int64
	res, err := tx.Exec(`
		INSERT INTO Riwayat_Kunjungan (id_rm, catatan)
		VALUES (?, ?)`,
		idRM, "",
	)
	if err != nil {
		err = fmt.Errorf("failed to insert Riwayat_Kunjungan: %v", err)
		return
	}
	idKunjungan, err = res.LastInsertId()
	if err != nil {
		err = fmt.Errorf("failed to get last insert id Riwayat_Kunjungan: %v", err)
		return
	}

	// 5. Insert Kunjungan_Poli
	_, err = tx.Exec(`
		INSERT INTO Kunjungan_Poli (id_poli, id_kunjungan)
		VALUES (?, ?)`,
		idPoli, idKunjungan,
	)
	if err != nil {
		err = fmt.Errorf("failed to insert into Kunjungan_Poli: %v", err)
		return
	}

	// 6. Hitung nomor antrian hari ini
	var maxNomor sql.NullInt64
	err = tx.QueryRow(`
		SELECT COALESCE(MAX(nomor_antrian),0)
		FROM Antrian
		WHERE id_poli=? AND DATE(created_at)=?`,
		idPoli, today,
	).Scan(&maxNomor)
	if err != nil {
		err = fmt.Errorf("failed to get max nomor antrian: %v", err)
		return
	}
	nomorAntrian = 1
	if maxNomor.Valid && maxNomor.Int64 > 0 {
		nomorAntrian = maxNomor.Int64 + 1
	}

	// 7. Ambil id_status “Menunggu”
	err = tx.QueryRow(`
		SELECT id_status
		FROM Status_Antrian
		WHERE status='Menunggu' LIMIT 1`,
	).Scan(&idStatus)
	if err != nil {
		err = fmt.Errorf("failed to get id_status for 'Menunggu': %v", err)
		return
	}

	// 8. Insert Antrian dengan nama_penanggung_jawab
	res, err = tx.Exec(`
		INSERT INTO Antrian
		  (id_pasien, id_poli, keluhan_utama, nomor_antrian,
		   id_status, priority_order, created_at, nama_penanggung_jawab)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), ?)`,
		idPasien, idPoli, keluhanUtama, nomorAntrian, idStatus, nomorAntrian, namaPenanggungJawab,
	)
	if err != nil {
		err = fmt.Errorf("failed to insert into Antrian: %v", err)
		return
	}
	idAntrian, err = res.LastInsertId()
	if err != nil {
		err = fmt.Errorf("failed to get id_antrian: %v", err)
		return
	}

	// 9. Insert Billing (tetap dilakukan, tanpa update RK)
	_, err = tx.Exec(`
		INSERT INTO Billing
		  (id_kunjungan, id_antrian, id_karyawan,
		   id_assessment, tipe_pembayaran,
		   total, id_status, created_at, updated_at)
		VALUES (?, ?, NULL, NULL, NULL, NULL, 1, NOW(), NOW())`,
		idKunjungan, idAntrian,
	)
	if err != nil {
		err = fmt.Errorf("failed to insert into Billing: %v", err)
		return
	}

	// 10. Update Riwayat_Kunjungan hanya dengan id_antrian
	_, err = tx.Exec(`
		UPDATE Riwayat_Kunjungan
		SET id_antrian = ?
		WHERE id_kunjungan = ?`,
		idAntrian, idKunjungan,
	)
	if err != nil {
		err = fmt.Errorf("failed to update Riwayat_Kunjungan: %v", err)
		return
	}

	// 11. Ambil nama_poli
	err = tx.QueryRow(`
		SELECT nama_poli
		FROM Poliklinik
		WHERE id_poli = ?`,
		idPoli,
	).Scan(&namaPoli)
	if err != nil {
		err = fmt.Errorf("failed to get nama_poli: %v", err)
		return
	}

	// Commit
	err = tx.Commit()
	return
}



func (s *PendaftaranService) GetAllPasienDataFiltered(namaFilter string, page, limit int) ([]map[string]interface{}, error) {
	// Base query untuk mengambil data pasien dengan join ke tabel Agama
	query := `
		SELECT p.id_pasien, p.nama, p.tanggal_lahir, p.jenis_kelamin, p.tempat_lahir, p.nik, 
		       p.kelurahan, p.kecamatan, p.kota_tinggal, p.alamat, p.no_telp, p.tanggal_regist,
		       p.pekerjaan, a.nama AS agama_nama, p.status_perkawinan
		FROM Pasien p
		LEFT JOIN Agama a ON p.id_agama = a.id_agama
	`
	conditions := []string{}
	args := []interface{}{}

	// Jika ada filter nama, tambahkan kondisi WHERE
	if strings.TrimSpace(namaFilter) != "" {
		conditions = append(conditions, "p.nama LIKE ?")
		args = append(args, "%"+namaFilter+"%")
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY p.tanggal_regist DESC"

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
		var nama, tanggalLahir, jenisKelamin, tempatLahir, nik, kelurahan, kecamatan, kotaTinggal, alamat, noTelp, tanggalRegist string
		var pekerjaan, agamaNama sql.NullString
		var statusPerkawinan sql.NullInt64

		if err := rows.Scan(&idPasien, &nama, &tanggalLahir, &jenisKelamin, &tempatLahir, &nik, &kelurahan, &kecamatan, 
			&kotaTinggal, &alamat, &noTelp, &tanggalRegist, &pekerjaan, &agamaNama, &statusPerkawinan); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}

		// Konversi status_perkawinan dari integer ke string
		var statusPerkawinanStr string
		if statusPerkawinan.Valid {
			if statusPerkawinan.Int64 == 1 {
				statusPerkawinanStr = "sudah kawin"
			} else if statusPerkawinan.Int64 == 0 {
				statusPerkawinanStr = "belum kawin"
			}
		} else {
			statusPerkawinanStr = ""
		}

		// Buat record dengan data tambahan
		record := map[string]interface{}{
			"ID_Pasien":        idPasien,
			"Nama":             nama,
			"Tanggal_Lahir":    tanggalLahir,
			"Jenis_Kelamin":    jenisKelamin,
			"Tempat_Lahir":     tempatLahir,
			"NIK":              nik,
			"Kelurahan":        kelurahan,
			"Kecamatan":        kecamatan,
			"Kota_Tinggal":     kotaTinggal,
			"Alamat":           alamat,
			"No_Telp":          noTelp,
			"Tanggal_Regist":   tanggalRegist,
			"Pekerjaan":        pekerjaan.String,
			"Agama":            agamaNama.String,
			"Status_Perkawinan": statusPerkawinanStr,
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
    result, err := s.DB.Exec(updateAntrianQuery, 7, idAntrian)
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

// GetDetailAntrianByID mengambil detail antrian + info pasien & dokter.
func (s *PendaftaranService) GetDetailAntrianByID(idAntrian int) (map[string]interface{}, error) {

	query := `
		SELECT
			p.id_pasien,
			p.nama,
			p.jenis_kelamin,
			p.tempat_lahir,
			DATE_FORMAT(p.tanggal_lahir,'%Y-%m-%d')               AS tanggal_lahir,
			p.nik,
			p.no_telp,
			p.alamat,
			p.kota_tinggal,
			p.kelurahan,
			p.kecamatan,
			IFNULL(ag.nama,'')                                     AS agama,
			IFNULL(p.pekerjaan,'')                                 AS pekerjaan,
			IFNULL(p.status_perkawinan,0)                          AS status_perkawinan,
			a.keluhan_utama,
			a.nama_penanggung_jawab,
			rm.id_rm,
			a.nomor_antrian
		FROM Antrian a
		JOIN Pasien        p  ON a.id_pasien = p.id_pasien
		JOIN Rekam_Medis   rm ON p.id_pasien = rm.id_pasien
		LEFT JOIN Agama    ag ON p.id_agama  = ag.id_agama
		WHERE a.id_antrian = ?
		ORDER BY rm.created_at DESC
		LIMIT 1
	`

	// ---------- scan ----------
	var (
		idPasien, nomorAntrian        int
		nama, jk, tmpLhr              string
		tglLhrStr, nik, telp          string
		alamat, kota, kel, kec        string
		agama, pekerjaan              string
		statusPK                      int          // 0/1
		keluhan, penanggungJawab      string
		idRM                          string
	)
	if err := s.DB.QueryRow(query, idAntrian).Scan(
		&idPasien, &nama, &jk, &tmpLhr,
		&tglLhrStr, &nik, &telp,
		&alamat, &kota, &kel, &kec,
		&agama, &pekerjaan, &statusPK,
		&keluhan, &penanggungJawab,
		&idRM, &nomorAntrian,
	); err != nil {
		return nil, fmt.Errorf("failed to get detail data: %v", err)
	}

	// ---------- umur (dalam hari) ----------
	tglLhr, _ := time.Parse("2006-01-02", tglLhrStr)
	umurHari := int(time.Since(tglLhr).Hours() / 24)

	// ---------- dokter terakhir ----------
	var namaDokter interface{} = nil
	var idAssessment int64
	err := s.DB.QueryRow(
		`SELECT id_assessment FROM Riwayat_Kunjungan
		  WHERE id_antrian = ? AND id_assessment IS NOT NULL
		  ORDER BY created_at DESC LIMIT 1`, idAntrian).Scan(&idAssessment)

	if err == nil { // ada assessment
		var idDokter int
		if err := s.DB.QueryRow(
			`SELECT id_karyawan FROM Assessment WHERE id_assessment = ?`, idAssessment).
			Scan(&idDokter); err != nil {
			return nil, fmt.Errorf("failed to get doctor id: %v", err)
		}
		var nd string
		if err := s.DB.QueryRow(
			`SELECT nama FROM Karyawan WHERE id_karyawan = ?`, idDokter).
			Scan(&nd); err != nil {
			return nil, fmt.Errorf("failed to get doctor name: %v", err)
		}
		namaDokter = nd
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get assessment record: %v", err)
	}

	statusPerkawinan := "Belum Menikah"
	if statusPK == 1 {
		statusPerkawinan = "Sudah Menikah"
	}

	// ---------- hasil ----------
	return map[string]interface{}{
		"id_antrian":        idAntrian,
		"nomor_antrian":     nomorAntrian,
		"id_pasien":         idPasien,
		"nama_pasien":       nama,
		"id_rm":             idRM,
		"jenis_kelamin":     jk,
		"tempat_lahir":      tmpLhr,
		"tanggal_lahir":     tglLhrStr,
		"umur":              umurHari,
		"nik":               nik,
		"no_telp":           telp,
		"alamat":            alamat,
		"kota":              kota,
		"kelurahan":         kel,
		"kecamatan":         kec,
		"agama":             agama,
		"pekerjaan":         pekerjaan,
		"status_perkawinan": statusPerkawinan,
		"keluhan_utama":     keluhan,
		"penanggung_jawab":  penanggungJawab,
		"nama_dokter":       namaDokter,
	}, nil
}



type Agama struct {
	IDAgama int    `json:"id_agama"`
	Nama    string `json:"nama"`
}
// GetAgamaList retrieves the list of religions from the Agama table
func (s *PendaftaranService) GetAgamaList() ([]Agama, error) {
	rows, err := s.DB.Query("SELECT id_agama, nama FROM Agama ORDER BY nama ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agamaList []Agama
	for rows.Next() {
		var agama Agama
		if err := rows.Scan(&agama.IDAgama, &agama.Nama); err != nil {
			return nil, err
		}
		agamaList = append(agamaList, agama)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return agamaList, nil
}