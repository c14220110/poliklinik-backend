package services

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type AntrianService struct {
	DB *sql.DB
}

func NewAntrianService(db *sql.DB) *AntrianService {
	return &AntrianService{DB: db}
}

func (s *AntrianService) MasukkanPasien(idPoli int) (map[string]interface{}, error) {
	// 1. Cari baris antrian teratas dengan id_status = 1 untuk id_poli yang diberikan dan untuk hari ini.
	query := `
		SELECT id_antrian 
		FROM Antrian 
		WHERE id_poli = ? AND id_status = 1 AND DATE(created_at) = CURDATE()
		ORDER BY nomor_antrian ASC 
		LIMIT 1
	`
	var idAntrian int
	err := s.DB.QueryRow(query, idPoli).Scan(&idAntrian)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tidak ada pasien dengan status 1 untuk poli dengan id %d pada hari ini", idPoli)
		}
		return nil, err
	}

	// 2. Update baris yang ditemukan, ubah id_status menjadi 3.
	updateQuery := `
		UPDATE Antrian 
		SET id_status = 3 
		WHERE id_antrian = ?
	`
	res, err := s.DB.Exec(updateQuery, idAntrian)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, fmt.Errorf("gagal mengupdate antrian, baris tidak ditemukan")
	}

	// 3. Ambil data detail pasien dan antrian.
	// Query ini menggabungkan data dari tabel Pasien, Rekam_Medis, dan Antrian.
	// Data yang diambil: id_pasien, nama, jenis_kelamin, tempat_lahir, tanggal_lahir, nik, no_telp, alamat,
	// kota_tinggal, kelurahan, kecamatan, id_rm, dan nomor_antrian.
	queryDetails := `
		SELECT p.id_pasien, p.nama, p.jenis_kelamin, p.tempat_lahir, 
		       DATE_FORMAT(p.tanggal_lahir, '%Y-%m-%d') AS tanggal_lahir, p.nik, p.no_telp, 
		       p.alamat, p.kota_tinggal, p.kelurahan, p.kecamatan, 
		       rm.id_rm, a.nomor_antrian
		FROM Antrian a
		JOIN Pasien p ON a.id_pasien = p.id_pasien
		JOIN Rekam_Medis rm ON p.id_pasien = rm.id_pasien
		WHERE a.id_antrian = ?
		ORDER BY rm.created_at DESC
		LIMIT 1
	`
	var idPasien int
	var nama, jenisKelamin, tempatLahir, tanggalLahirStr, nik, noTelp, alamat, kotaTinggal, kelurahan, kecamatan string
	var idRM string
	var nomorAntrian int

	err = s.DB.QueryRow(queryDetails, idAntrian).Scan(
		&idPasien, &nama, &jenisKelamin, &tempatLahir, &tanggalLahirStr, &nik, &noTelp,
		&alamat, &kotaTinggal, &kelurahan, &kecamatan, &idRM, &nomorAntrian,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get detail data: %v", err)
	}

	// Parse tanggal_lahir dengan layout "2006-01-02"
	tanggalLahir, err := time.Parse("2006-01-02", tanggalLahirStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tanggal_lahir: %v", err)
	}

	// Hitung umur
	now := time.Now()
	umur := now.Year() - tanggalLahir.Year()
	if now.YearDay() < tanggalLahir.YearDay() {
		umur--
	}

	result := map[string]interface{}{
		"id_antrian":    idAntrian,
		"nomor_antrian": nomorAntrian,
		"id_pasien":     idPasien,
		"nama_pasien":   nama,
		"id_rm":         idRM,
		"jenis_kelamin": jenisKelamin,
		"tempat_lahir":  tempatLahir,
		"tanggal_lahir": tanggalLahirStr,
		"nik":           nik,
		"no_telp":       noTelp,
		"alamat":        alamat,
		"kota":          kotaTinggal, // Disesuaikan dengan kolom 'kota_tinggal'
		"kelurahan":     kelurahan,
		"kecamatan":     kecamatan,
		"umur":          umur,
	}

	return result, nil
}




// GetAntrianTerlama mengambil ID_Antrian dan Nomor_Antrian dari pasien dengan antrian paling lama (status = 1) pada hari ini
func (s *AntrianService) GetAntrianTerlama(idPoli int) (map[string]interface{}, error) {
	query := `
		SELECT id_antrian, nomor_antrian 
		FROM Antrian 
		WHERE id_poli = ? AND id_status = 1 AND DATE(created_at) = CURDATE()
		ORDER BY nomor_antrian ASC 
		LIMIT 1
	`
	var idAntrian int
	var nomorAntrian int

	err := s.DB.QueryRow(query, idPoli).Scan(&idAntrian, &nomorAntrian)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("tidak ada antrian yang menunggu")
		}
		return nil, err
	}

	result := map[string]interface{}{
		"id_antrian":    idAntrian,
		"nomor_antrian": nomorAntrian,
	}

	return result, nil
}

func (s *AntrianService) MasukkanPasienKeDokter(idPoli int) (map[string]interface{}, error) {
	// 1. Cari baris antrian teratas dengan id_status = 4 untuk id_poli yang diberikan dan untuk hari ini.
	query := `
		SELECT id_antrian 
		FROM Antrian 
		WHERE id_poli = ? AND id_status = 4 AND DATE(created_at) = CURDATE()
		ORDER BY nomor_antrian ASC 
		LIMIT 1
	`
	var idAntrian int
	err := s.DB.QueryRow(query, idPoli).Scan(&idAntrian)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tidak ada pasien dengan status pra-konsul untuk poli dengan id %d pada hari ini", idPoli)
		}
		return nil, err
	}

	// 2. Update baris yang ditemukan, ubah id_status menjadi 5.
	updateQuery := `
		UPDATE Antrian 
		SET id_status = 5 
		WHERE id_antrian = ?
	`
	res, err := s.DB.Exec(updateQuery, idAntrian)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, fmt.Errorf("gagal mengupdate antrian, baris tidak ditemukan")
	}

	// 3. Ambil data tambahan: id_pasien, nama pasien, jenis_kelamin, id_rm, tanggal_lahir, dan nomor_antrian
	queryDetails := `
		SELECT p.id_pasien, p.nama, p.jenis_kelamin, rm.id_rm, p.tanggal_lahir, a.nomor_antrian
		FROM Antrian a
		JOIN Pasien p ON a.id_pasien = p.id_pasien
		JOIN Rekam_Medis rm ON p.id_pasien = rm.id_pasien
		WHERE a.id_antrian = ?
		ORDER BY rm.created_at DESC
		LIMIT 1
	`
	var idPasien int
	var nama, jenisKelamin, tanggalLahirStr string
	var idRM string
	var nomorAntrian int

	err = s.DB.QueryRow(queryDetails, idAntrian).Scan(&idPasien, &nama, &jenisKelamin, &idRM, &tanggalLahirStr, &nomorAntrian)
	if err != nil {
		return nil, fmt.Errorf("failed to get detail data: %v", err)
	}

	// Parse tanggal_lahir; gunakan layout RFC3339 karena format di database misalnya "1995-08-15T00:00:00+07:00"
	tanggalLahir, err := time.Parse(time.RFC3339, tanggalLahirStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tanggal_lahir: %v", err)
	}

	// Hitung umur
	now := time.Now()
	umur := now.Year() - tanggalLahir.Year()
	if now.YearDay() < tanggalLahir.YearDay() {
		umur--
	}

	result := map[string]interface{}{
		"id_antrian":     idAntrian,
		"id_pasien":      idPasien,
		"nama_pasien":    nama,
		"jenis_kelamin":  jenisKelamin,
		"id_rm":          idRM,
		"nomor_antrian":  nomorAntrian,
		"umur":           umur,
	}

	return result, nil
}

func (s *AntrianService) PulangkanPasien(idAntrian int) error {
	// Periksa status saat ini
	var currentStatus int
	checkQuery := "SELECT id_status FROM Antrian WHERE id_antrian = ?"
	err := s.DB.QueryRow(checkQuery, idAntrian).Scan(&currentStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("antrian dengan id %d tidak ditemukan", idAntrian)
		}
		return fmt.Errorf("gagal memeriksa status antrian: %v", err)
	}
	if currentStatus != 5 {
		return fmt.Errorf("status antrian saat ini bukan Konsultasi (5), melainkan %d", currentStatus)
	}

	// Update status ke 6 (Pulang)
	updateQuery := "UPDATE Antrian SET id_status = ? WHERE id_antrian = ?"
	result, err := s.DB.Exec(updateQuery, 6, idAntrian)
	if err != nil {
		return fmt.Errorf("gagal mengupdate antrian: %v", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa update antrian: %v", err)
	}
	if affected == 0 {
		return fmt.Errorf("tidak ada baris yang terupdate, antrian dengan id %d mungkin tidak ada", idAntrian)
	}
	return nil
}

// AlihkanPasien mengubah status antrian menjadi 4 untuk id_antrian yang diberikan.
func (s *AntrianService) AlihkanPasien(idAntrian int) error {
	updateQuery := "UPDATE Antrian SET id_status = ? WHERE id_antrian = ?"
	result, err := s.DB.Exec(updateQuery, 4, idAntrian)
	if err != nil {
		return fmt.Errorf("gagal mengupdate antrian: %v", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa update antrian: %v", err)
	}
	if affected == 0 {
		return fmt.Errorf("tidak ada baris yang terupdate, antrian dengan id %d mungkin tidak ada", idAntrian)
	}
	return nil
}


// GetTodayScreeningAntrianByPoli mengembalikan daftar antrian status=3
// untuk hari ini pada poliklinik tertentu, diurutkan berdasarkan
// priority_order terendah (non‑NULL) lebih dulu, lalu id_antrian.
func (s *AntrianService) GetTodayScreeningAntrianByPoli(idPoli int) ([]map[string]interface{}, error) {
    query := `
        SELECT
            a.id_antrian,
            a.id_pasien,
            p.nama,
            a.priority_order
        FROM Antrian a
        JOIN Pasien p ON a.id_pasien = p.id_pasien
        WHERE a.id_poli   = ?
          AND a.id_status = 3
          AND DATE(a.created_at) = CURDATE()
        ORDER BY
          CASE WHEN a.priority_order IS NULL THEN 1 ELSE 0 END,
          a.priority_order,
          a.id_antrian
    `

    rows, err := s.DB.Query(query, idPoli)
    if err != nil {
        return nil, fmt.Errorf("query error: %v", err)
    }
    defer rows.Close()

    var results []map[string]interface{}

    for rows.Next() {
        var (
            idAntrian, idPasien int
            nama               string
            priority           sql.NullInt64
        )
        if err := rows.Scan(&idAntrian, &idPasien, &nama, &priority); err != nil {
            return nil, fmt.Errorf("scan error: %v", err)
        }

        record := map[string]interface{}{
            "id_antrian":     idAntrian,
            "id_pasien":      idPasien,
            "nama_pasien":    nama,
            "priority_order": nil,
        }
        if priority.Valid {
            record["priority_order"] = priority.Int64
        }
        results = append(results, record)
    }

    if err = rows.Err(); err != nil {
        return nil, err
    }
    return results, nil
}


// GetDetailAntrianByID mengembalikan detail antrian + biodata pasien
func (s *AntrianService) GetDetailAntrianByID(idAntrian int) (map[string]interface{}, error) {

	query := `
	SELECT
		p.id_pasien,
		p.nama,
		p.jenis_kelamin,
		p.tempat_lahir,
		DATE_FORMAT(p.tanggal_lahir,'%Y-%m-%d')        AS tanggal_lahir,
		p.nik,
		p.no_telp,
		p.alamat,
		p.kota_tinggal,
		p.kelurahan,
		p.kecamatan,
		IFNULL(ag.nama,'')                             AS agama,      -- ← diubah
		IFNULL(p.pekerjaan,'')                         AS pekerjaan,
		IFNULL(p.status_perkawinan,0)                  AS status_perkawinan,
		a.keluhan_utama,
		a.nama_penanggung_jawab,
		rm.id_rm,
		a.nomor_antrian
	FROM Antrian a
	JOIN Pasien      p  ON a.id_pasien = p.id_pasien
	JOIN Rekam_Medis rm ON p.id_pasien = rm.id_pasien
	LEFT JOIN Agama  ag ON p.id_agama  = ag.id_agama
	WHERE a.id_antrian = ?
	ORDER BY rm.created_at DESC
	LIMIT 1
`


	var (
		idPasien, nomorAntrian     int
		nama, jk, tmpLahir         string
		tglLahirStr, nik, telp     string
		alamat, kota, kel, kec     string
		agama, pekerjaan           string
		statusPK                   int           // 0/1
		keluhan, pjName            string
		idRM                       string
	)

	err := s.DB.QueryRow(query, idAntrian).Scan(
		&idPasien, &nama, &jk, &tmpLahir,
		&tglLahirStr, &nik, &telp,
		&alamat, &kota, &kel, &kec,
		&agama, &pekerjaan, &statusPK,
		&keluhan, &pjName,
		&idRM, &nomorAntrian,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get detail data: %v", err)
	}

	// Hitung umur
	tglLahir, _ := time.Parse("2006-01-02", tglLahirStr)
	now := time.Now()
	umur := now.Year() - tglLahir.Year()
	if now.YearDay() < tglLahir.YearDay() { umur-- }

	// Konversi status_perkawinan
	var statusPerkawinan string
	if statusPK == 1 {
		statusPerkawinan = "Sudah Menikah"
	} else {
		statusPerkawinan = "Belum Menikah"
	}

	return map[string]interface{}{
		"id_antrian":        idAntrian,
		"nomor_antrian":     nomorAntrian,
		"id_pasien":         idPasien,
		"nama_pasien":       nama,
		"id_rm":             idRM,
		"jenis_kelamin":     jk,
		"tempat_lahir":      tmpLahir,
		"tanggal_lahir":     tglLahirStr,
		"umur":              umur,
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
		"penanggung_jawab":  pjName,
	}, nil
}
