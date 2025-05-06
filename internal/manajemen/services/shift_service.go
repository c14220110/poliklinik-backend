package services

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ShiftService struct {
	DB *sql.DB
}

func NewShiftService(db *sql.DB) *ShiftService {
	return &ShiftService{DB: db}
}


func (s *ShiftService) AssignShift(idPoli, idKaryawan, idRole, idShift, idManagement int, tanggalStr string) (int64, error) {
	// 0. Cek apakah karyawan memiliki role sesuai parameter (dilakukan di luar transaksi).
	var roleCount int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM Detail_Role_Karyawan WHERE id_karyawan = ? AND id_role = ?", idKaryawan, idRole).Scan(&roleCount)
	if err != nil {
		return 0, fmt.Errorf("failed to check role for karyawan: %v", err)
	}
	if roleCount == 0 {
		return 0, fmt.Errorf("karyawan dengan id %d tidak memiliki role %d", idKaryawan, idRole)
	}

	// 1. Validasi format tanggal
	_, err = time.Parse("2006-01-02", tanggalStr)
	if err != nil {
		return 0, fmt.Errorf("format tanggal tidak valid: %v", err)
	}

	// Mulai transaksi
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}

	// 2. Cek apakah karyawan sudah memiliki shift yang sama di poli pada tanggal ini
	var existingCount int
	err = tx.QueryRow(
		"SELECT COUNT(*) FROM Shift_Karyawan WHERE id_karyawan = ? AND id_poli = ? AND id_shift = ? AND tanggal = ?",
		idKaryawan, idPoli, idShift, tanggalStr,
	).Scan(&existingCount)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to check existing shift: %v", err)
	}
	if existingCount > 0 {
		tx.Rollback()
		return 0, fmt.Errorf("User dengan role %d sudah memiliki shift %d di poli %d pada tanggal %s", idRole, idShift, idPoli, tanggalStr)
	}

	// 3. Ambil data Shift untuk mendapatkan jam_mulai dan jam_selesai default
	var jamMulai, jamSelesai string
	queryShift := "SELECT jam_mulai, jam_selesai FROM Shift WHERE id_shift = ?"
	err = tx.QueryRow(queryShift, idShift).Scan(&jamMulai, &jamSelesai)
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("id_shift %d tidak ditemukan", idShift)
		}
		return 0, err
	}

	// 4. Insert ke tabel Shift_Karyawan
	insertQuery := `
		INSERT INTO Shift_Karyawan (id_poli, id_shift, id_karyawan, custom_jam_mulai, custom_jam_selesai, tanggal)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	res, err := tx.Exec(insertQuery, idPoli, idShift, idKaryawan, jamMulai, jamSelesai, tanggalStr)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	idShiftKaryawan, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// 5. Insert ke tabel Management_Shift_Karyawan
	insertManagementShiftQuery := `
		INSERT INTO Management_Shift_Karyawan (id_management, id_shift_karyawan, created_by, updated_by, deleted_by)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(insertManagementShiftQuery, idManagement, idShiftKaryawan, idManagement, idManagement, 0)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Commit transaksi
	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return idShiftKaryawan, nil
}


func (s *ShiftService) UpdateCustomShift(idShiftKaryawan int, newCustomMulai, newCustomSelesai string) error {
	// Parse waktu custom yang baru dengan format "15:04:05"
	newMulai, err := time.Parse("15:04:05", newCustomMulai)
	if err != nil {
		return fmt.Errorf("format custom_jam_mulai tidak valid: %v", err)
	}
	newSelesai, err := time.Parse("15:04:05", newCustomSelesai)
	if err != nil {
		return fmt.Errorf("format custom_jam_selesai tidak valid: %v", err)
	}

	// Ambil default waktu shift dari tabel Shift berdasarkan id_shift dari Shift_Karyawan
	var shiftJamMulai, shiftJamSelesai string
	query := `
		SELECT s.jam_mulai, s.jam_selesai 
		FROM Shift_Karyawan sk
		JOIN Shift s ON sk.id_shift = s.id_shift
		WHERE sk.id_shift_karyawan = ?
	`
	err = s.DB.QueryRow(query, idShiftKaryawan).Scan(&shiftJamMulai, &shiftJamSelesai)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("record Shift_Karyawan dengan id %d tidak ditemukan", idShiftKaryawan)
		}
		return err
	}

	// Parse default waktu shift
	defaultMulai, err := time.Parse("15:04:05", shiftJamMulai)
	if err != nil {
		return fmt.Errorf("format default jam_mulai tidak valid: %v", err)
	}
	defaultSelesai, err := time.Parse("15:04:05", shiftJamSelesai)
	if err != nil {
		return fmt.Errorf("format default jam_selesai tidak valid: %v", err)
	}

	// Validasi: custom_jam_mulai tidak boleh sebelum default dan custom_jam_selesai tidak boleh melewati default
	if newMulai.Before(defaultMulai) || newSelesai.After(defaultSelesai) {
		return fmt.Errorf("custom shift harus berada dalam rentang waktu %s - %s", shiftJamMulai, shiftJamSelesai)
	}

	// Update record Shift_Karyawan dengan waktu custom yang baru
	updateQuery := `
		UPDATE Shift_Karyawan 
		SET custom_jam_mulai = ?, custom_jam_selesai = ?
		WHERE id_shift_karyawan = ?
	`
	res, err := s.DB.Exec(updateQuery, newCustomMulai, newCustomSelesai, idShiftKaryawan)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("tidak ada record yang diupdate")
	}
	return nil
}

func (s *ShiftService) SoftDeleteShiftKaryawan(idShiftKaryawan int, idManagement int) error {
	// Update deleted_by dari NULL menjadi idManagement untuk record yang belum dihapus
	updateQuery := `
		UPDATE Management_Shift_Karyawan 
		SET deleted_by = ? 
		WHERE id_shift_karyawan = ? AND deleted_by IS NULL
	`
	res, err := s.DB.Exec(updateQuery, idManagement, idShiftKaryawan)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tidak ada record yang di soft delete atau record sudah di soft delete")
	}
	return nil
}

func (s *ShiftService) GetShiftPoliList(idPoliFilter string) ([]map[string]interface{}, error) {
	// Kondisi join hanya berdasarkan id_shift dan tanggal, tanpa filter CURTIME()
	joinCondition := "s.id_shift = sk.id_shift AND sk.tanggal = CURDATE()"
	var args []interface{}

	// Jika ada filter berdasarkan id_poli, tambahkan kondisi
	if idPoliFilter != "" {
		joinCondition += " AND sk.id_poli = ?"
		args = append(args, idPoliFilter)
	}

	query := fmt.Sprintf(`
		SELECT 
			s.id_shift, 
			s.jam_mulai, 
			s.jam_selesai,
			CASE 
				WHEN s.id_shift = 1 THEN 'Shift Pagi'
				WHEN s.id_shift = 2 THEN 'Shift Sore'
				ELSE 'Shift Lainnya'
			END AS nama_shift,
			COUNT(DISTINCT sk.id_karyawan) AS jumlah_tenkes
		FROM Shift s
		LEFT JOIN Shift_Karyawan sk 
			ON %s
		GROUP BY s.id_shift, s.jam_mulai, s.jam_selesai
		ORDER BY s.id_shift
	`, joinCondition)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var idShift int
		var jamMulai, jamSelesai, namaShift string
		var jumlahTenkes int
		if err := rows.Scan(&idShift, &jamMulai, &jamSelesai, &namaShift, &jumlahTenkes); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}
		record := map[string]interface{}{
			"id_shift":      idShift,
			"nama_shift":    namaShift,
			"jam_mulai":     jamMulai,
			"jam_selesai":   jamSelesai,
			"jumlah_tenkes": jumlahTenkes,
		}
		results = append(results, record)
	}
	return results, nil
}

// GetListKaryawanFiltered mengambil daftar karyawan berdasarkan:
// - id_poli (wajib)
// - id_shift (wajib)
// - id_role (opsional; jika tidak ada, tampilkan semua)
// - tanggal (opsional; jika kosong, hari ini)
func (s *ShiftService) GetListKaryawanFiltered(
	idPoliFilter, idShiftFilter, idRoleFilter, tanggalFilter string,
) ([]map[string]interface{}, error) {
	// Parse id_poli dan id_shift (wajib)
	idPoli, err := strconv.Atoi(idPoliFilter)
	if err != nil {
		return nil, fmt.Errorf("invalid id_poli value: %v", err)
	}
	idShift, err := strconv.Atoi(idShiftFilter)
	if err != nil {
		return nil, fmt.Errorf("invalid id_shift value: %v", err)
	}

	// Parse tanggal: jika kosong, gunakan hari ini; jika ada, format DD/MM/YYYY
	var tanggal string
	if strings.TrimSpace(tanggalFilter) == "" {
		tanggal = time.Now().Format("2006-01-02")
	} else {
		parsed, err := time.Parse("02/01/2006", tanggalFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid tanggal format, harus DD/MM/YYYY: %v", err)
		}
		tanggal = parsed.Format("2006-01-02")
	}

	// Jika filter id_role disediakan
	if strings.TrimSpace(idRoleFilter) != "" {
		idRole, err := strconv.Atoi(idRoleFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid id_role value: %v", err)
		}
		
		query := `
		SELECT
			k.id_karyawan,
			k.nama,
			k.nik,
			k.username,
			k.no_telp,
			k.alamat,
			k.jenis_kelamin,
			sk.custom_jam_mulai,
			sk.custom_jam_selesai,
			sk.id_shift_karyawan,
			r.nama_role
		FROM Karyawan k
		JOIN Shift_Karyawan sk
			ON k.id_karyawan = sk.id_karyawan
			AND sk.id_poli = ?
			AND sk.id_shift = ?
			AND sk.tanggal = ?
		JOIN Detail_Role_Karyawan drk
			ON k.id_karyawan = drk.id_karyawan
			AND drk.id_role = ?
		JOIN Role r
			ON drk.id_role = r.id_role
		ORDER BY k.id_karyawan
		`
		args := []interface{}{idPoli, idShift, tanggal, idRole}
		rows, err := s.DB.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("query error: %v", err)
		}
		defer rows.Close()

		var results []map[string]interface{}
		for rows.Next() {
			var (
				idKaryawan       int
				nama, nik, username, noTelp, alamat, jenisKelamin string
				customJamMulai, customJamSelesai                string
				idShiftKaryawan                                 int
				namaRole                                        string
			)
			if err := rows.Scan(
				&idKaryawan, &nama, &nik, &username, &noTelp,
				&alamat, &jenisKelamin, &customJamMulai,
				&customJamSelesai, &idShiftKaryawan, &namaRole,
			); err != nil {
				return nil, fmt.Errorf("scan error: %v", err)
			}
			record := map[string]interface{}{
				"id_karyawan":        idKaryawan,
				"nama":               nama,
				"NIK":                nik,
				"username":           username,
				"role":               namaRole,
				"no_telp":            noTelp,
				"alamat":             alamat,
				"jenis_kelamin":      jenisKelamin,
				"custom_jam_mulai":   customJamMulai,
				"custom_jam_selesai": customJamSelesai,
				"id_shift_karyawan":  idShiftKaryawan,
			}
			results = append(results, record)
		}
		return results, nil
	}

	// Tanpa filter id_role: agregasi semua nama_role per karyawan
	query := `
		SELECT
			k.id_karyawan,
			k.nama,
			k.nik,
			k.username,
			k.no_telp,
			k.alamat,
			k.jenis_kelamin,
			sk.custom_jam_mulai,
			sk.custom_jam_selesai,
			sk.id_shift_karyawan,
			GROUP_CONCAT(r.nama_role SEPARATOR ',') AS roles
		FROM Karyawan k
		JOIN Shift_Karyawan sk
			ON k.id_karyawan = sk.id_karyawan
			AND sk.id_poli = ?
			AND sk.id_shift = ?
			AND sk.tanggal = ?
		LEFT JOIN Detail_Role_Karyawan drk
			ON k.id_karyawan = drk.id_karyawan
		LEFT JOIN Role r
			ON drk.id_role = r.id_role
		GROUP BY
			k.id_karyawan, k.nama, k.nik, k.username,
			k.no_telp, k.alamat, k.jenis_kelamin,
			sk.custom_jam_mulai, sk.custom_jam_selesai,
			sk.id_shift_karyawan
		ORDER BY k.id_karyawan
		`
	args := []interface{}{idPoli, idShift, tanggal}
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var (
			idKaryawan       int
			nama, nik, username, noTelp, alamat, jenisKelamin string
			customJamMulai, customJamSelesai                string
			idShiftKaryawan                                 int
			rolesStr                                       sql.NullString
		)
		if err := rows.Scan(
			&idKaryawan, &nama, &nik, &username, &noTelp,
			&alamat, &jenisKelamin, &customJamMulai,
			&customJamSelesai, &idShiftKaryawan, &rolesStr,
		); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}

		roleOutput := ""
		if rolesStr.Valid {
			roleOutput = rolesStr.String
		}
		record := map[string]interface{}{
			"id_karyawan":        idKaryawan,
			"nama":               nama,
			"NIK":                nik,
			"username":           username,
			"role":               roleOutput,
			"no_telp":            noTelp,
			"alamat":             alamat,
			"jenis_kelamin":      jenisKelamin,
			"custom_jam_mulai":   customJamMulai,
			"custom_jam_selesai": customJamSelesai,
			"id_shift_karyawan":  idShiftKaryawan,
		}
		results = append(results, record)
	}
	return results, nil
}


func (s *ShiftService) GetKaryawanTanpaShift(idShift int, idRole *int, tanggalStr string, idPoli int) ([]map[string]interface{}, error) {
    // 1. Tentukan tanggal
    var tanggal time.Time
    if tanggalStr == "" {
        tanggal = time.Now() // Gunakan hari ini jika kosong
    } else {
        var err error
        tanggal, err = time.Parse("02/01/2006", tanggalStr) // Parse format DD/MM/YYYY
        if err != nil {
            return nil, fmt.Errorf("format tanggal tidak valid, gunakan DD/MM/YYYY")
        }
    }
    tanggalFormatted := tanggal.Format("2006-01-02") // Format YYYY-MM-DD untuk SQL

    // 2. Buat query SQL untuk mengambil semua role per karyawan
    query := `
        SELECT k.id_karyawan, k.nama, k.jenis_kelamin, k.no_telp, s.jam_mulai, s.jam_selesai,
               GROUP_CONCAT(DISTINCT r.nama_role SEPARATOR ', ') AS roles
        FROM Karyawan k
        INNER JOIN Detail_Role_Karyawan drk ON k.id_karyawan = drk.id_karyawan
        INNER JOIN Role r ON drk.id_role = r.id_role
        LEFT JOIN Shift_Karyawan sk ON k.id_karyawan = sk.id_karyawan 
            AND sk.id_poli = ? 
            AND sk.id_shift = ? 
            AND sk.tanggal = ?
        INNER JOIN Shift s ON s.id_shift = ?
        WHERE sk.id_shift_karyawan IS NULL
        GROUP BY k.id_karyawan, k.nama, k.jenis_kelamin, k.no_telp, s.jam_mulai, s.jam_selesai
    `
    args := []interface{}{idPoli, idShift, tanggalFormatted, idShift}

    // 3. Tambahkan filter id_role jika ada
    if idRole != nil {
        var roleName string
        switch *idRole {
        case 1:
            roleName = "Administrasi"
        case 2:
            roleName = "Suster"
        case 3:
            roleName = "Dokter"
        default:
            return nil, fmt.Errorf("id_role %d tidak valid", *idRole)
        }
        query += " HAVING GROUP_CONCAT(DISTINCT r.nama_role SEPARATOR ', ') LIKE ?"
        args = append(args, fmt.Sprintf("%%%s%%", roleName))
    }

    // 4. Eksekusi query
    rows, err := s.DB.Query(query, args...)
    if err != nil {
        return nil, fmt.Errorf("gagal mengambil data karyawan: %v", err)
    }
    defer rows.Close()

    // 5. Simpan hasil sementara
    var tempResults []struct {
        idKaryawan   int
        nama         string
        jenisKelamin string
        noTelp       string
        jamMulai     string
        jamSelesai   string
        rolesStr     string
    }
    for rows.Next() {
        var temp struct {
            idKaryawan   int
            nama         string
            jenisKelamin string
            noTelp       string
            jamMulai     string
            jamSelesai   string
            rolesStr     sql.NullString
        }
        if err := rows.Scan(&temp.idKaryawan, &temp.nama, &temp.jenisKelamin, &temp.noTelp, &temp.jamMulai, &temp.jamSelesai, &temp.rolesStr); err != nil {
            return nil, fmt.Errorf("gagal membaca data karyawan: %v", err)
        }
        var rolesStr string
        if temp.rolesStr.Valid {
            rolesStr = temp.rolesStr.String
        }
        tempResults = append(tempResults, struct {
            idKaryawan   int
            nama         string
            jenisKelamin string
            noTelp       string
            jamMulai     string
            jamSelesai   string
            rolesStr     string
        }{temp.idKaryawan, temp.nama, temp.jenisKelamin, temp.noTelp, temp.jamMulai, temp.jamSelesai, rolesStr})
    }

    // 6. Proses pengelompokan role
    var results []map[string]interface{}
    for _, tr := range tempResults {
        roles := strings.Split(tr.rolesStr, ", ")
        hasAdminSuster := false
        hasDokter := false
        var otherRoles []string

        for _, role := range roles {
            trimmedRole := strings.TrimSpace(role)
            switch trimmedRole {
            case "Administrasi":
                hasAdminSuster = true
                otherRoles = append(otherRoles, "Admin")
            case "Suster":
                hasAdminSuster = true
                otherRoles = append(otherRoles, "Suster")
            case "Dokter":
                hasDokter = true
            }
        }

        // Gabungkan Admin dan Suster jika ada
        if hasAdminSuster && len(otherRoles) > 0 {
            record := map[string]interface{}{
                "id_karyawan":   tr.idKaryawan,
                "nama":          tr.nama,
                "jenis_kelamin": tr.jenisKelamin,
                "roles":         strings.Join(otherRoles, ", "),
                "nomor_telepon": tr.noTelp,
                "jam_mulai":     tr.jamMulai,
                "jam_akhir":     tr.jamSelesai,
            }
            results = append(results, record)
        }
        // Tambahkan record terpisah untuk Dokter jika ada
        if hasDokter {
            record := map[string]interface{}{
                "id_karyawan":   tr.idKaryawan,
                "nama":          tr.nama,
                "jenis_kelamin": tr.jenisKelamin,
                "roles":         "Dokter",
                "nomor_telepon": tr.noTelp,
                "jam_mulai":     tr.jamMulai,
                "jam_akhir":     tr.jamSelesai,
            }
            results = append(results, record)
        }
    }

    return results, nil
}

type AssignShiftRequest struct {
    IDKaryawan int    `json:"id_karyawan"`
    IDRole     int    `json:"id_role"`
    JamMulai   string `json:"jam_mulai"`
    JamAkhir   string `json:"jam_akhir"`
}

func (s *ShiftService) AssignShiftNew(idPoli, idShift, idManagement int, tanggalStr string, requests []AssignShiftRequest) error {
    // 1. Validasi format tanggal DD/MM/YYYY
    tanggal, err := time.Parse("02/01/2006", tanggalStr)
    if err != nil {
        return fmt.Errorf("format tanggal tidak valid, gunakan DD/MM/YYYY: %v", err)
    }
    tanggalFormatted := tanggal.Format("2006-01-02") // Format untuk SQL: YYYY-MM-DD

    // Mulai transaksi
    tx, err := s.DB.Begin()
    if err != nil {
        return fmt.Errorf("gagal memulai transaksi: %v", err)
    }

    // 2. Ambil jam_mulai dan jam_selesai dari tabel Shift untuk id_shift
    var shiftJamMulai, shiftJamSelesai string
    err = tx.QueryRow("SELECT jam_mulai, jam_selesai FROM Shift WHERE id_shift = ?", idShift).Scan(&shiftJamMulai, &shiftJamSelesai)
    if err != nil {
        tx.Rollback()
        if err == sql.ErrNoRows {
            return fmt.Errorf("id_shift %d tidak ditemukan", idShift)
        }
        return fmt.Errorf("gagal mengambil data shift: %v", err)
    }

    // Parse jam dari tabel Shift ke time.Time
    shiftStart, err := time.Parse("15:04:05", shiftJamMulai)
    if err != nil {
        tx.Rollback()
        return fmt.Errorf("format jam_mulai pada tabel Shift tidak valid: %v", err)
    }
    shiftEnd, err := time.Parse("15:04:05", shiftJamSelesai)
    if err != nil {
        tx.Rollback()
        return fmt.Errorf("format jam_selesai pada tabel Shift tidak valid: %v", err)
    }

    // Proses setiap request
    for _, req := range requests {
        // 3. Validasi format jam_mulai dan jam_akhir
        customStart, err := time.Parse("15:04:05", req.JamMulai)
        if err != nil {
            tx.Rollback()
            return fmt.Errorf("format jam_mulai untuk karyawan %d tidak valid: %v", req.IDKaryawan, err)
        }
        customEnd, err := time.Parse("15:04:05", req.JamAkhir)
        if err != nil {
            tx.Rollback()
            return fmt.Errorf("format jam_akhir untuk karyawan %d tidak valid: %v", req.IDKaryawan, err)
        }

        // 4. Validasi rentang jam
        if customStart.Before(shiftStart) || customStart.After(shiftEnd) {
            tx.Rollback()
            return fmt.Errorf("jam_mulai %s untuk karyawan %d harus dalam rentang %s - %s", req.JamMulai, req.IDKaryawan, shiftJamMulai, shiftJamSelesai)
        }
        if customEnd.Before(shiftStart) || customEnd.After(shiftEnd) {
            tx.Rollback()
            return fmt.Errorf("jam_akhir %s untuk karyawan %d harus dalam rentang %s - %s", req.JamAkhir, req.IDKaryawan, shiftJamMulai, shiftJamSelesai)
        }
        if customEnd.Before(customStart) {
            tx.Rollback()
            return fmt.Errorf("jam_akhir %s untuk karyawan %d tidak boleh sebelum jam_mulai %s", req.JamAkhir, req.IDKaryawan, req.JamMulai)
        }

        // 5. Cek apakah karyawan memiliki role yang sesuai
        var roleCount int
        err = tx.QueryRow(
            "SELECT COUNT(*) FROM Detail_Role_Karyawan WHERE id_karyawan = ? AND id_role = ?",
            req.IDKaryawan, req.IDRole,
        ).Scan(&roleCount)
        if err != nil {
            tx.Rollback()
            return fmt.Errorf("gagal memeriksa role untuk karyawan %d: %v", req.IDKaryawan, err)
        }
        if roleCount == 0 {
            tx.Rollback()
            return fmt.Errorf("karyawan dengan id %d tidak memiliki role %d", req.IDKaryawan, req.IDRole)
        }

        // 6. Cek apakah karyawan sudah memiliki shift yang sama di poli pada tanggal ini
        var existingCount int
        err = tx.QueryRow(
            "SELECT COUNT(*) FROM Shift_Karyawan WHERE id_karyawan = ? AND id_poli = ? AND id_shift = ? AND tanggal = ?",
            req.IDKaryawan, idPoli, idShift, tanggalFormatted,
        ).Scan(&existingCount)
        if err != nil {
            tx.Rollback()
            return fmt.Errorf("gagal memeriksa shift yang ada untuk karyawan %d: %v", req.IDKaryawan, err)
        }
        if existingCount > 0 {
            tx.Rollback()
            return fmt.Errorf("karyawan %d sudah memiliki shift %d di poli %d pada tanggal %s", req.IDKaryawan, idShift, idPoli, tanggalStr)
        }

        // 7. Insert ke tabel Shift_Karyawan dengan jam custom dari request
        insertQuery := `
            INSERT INTO Shift_Karyawan (id_poli, id_shift, id_karyawan, custom_jam_mulai, custom_jam_selesai, tanggal)
            VALUES (?, ?, ?, ?, ?, ?)
        `
        res, err := tx.Exec(insertQuery, idPoli, idShift, req.IDKaryawan, req.JamMulai, req.JamAkhir, tanggalFormatted)
        if err != nil {
            tx.Rollback()
            return fmt.Errorf("gagal memasukkan shift untuk karyawan %d: %v", req.IDKaryawan, err)
        }
        idShiftKaryawan, err := res.LastInsertId()
        if err != nil {
            tx.Rollback()
            return fmt.Errorf("gagal mendapatkan id_shift_karyawan untuk karyawan %d: %v", req.IDKaryawan, err)
        }

        // 8. Insert ke tabel Management_Shift_Karyawan
        insertManagementShiftQuery := `
            INSERT INTO Management_Shift_Karyawan (id_management, id_shift_karyawan, created_by, updated_by, deleted_by)
            VALUES (?, ?, ?, ?, ?)
        `
        _, err = tx.Exec(insertManagementShiftQuery, idManagement, idShiftKaryawan, idManagement, idManagement, 0)
        if err != nil {
            tx.Rollback()
            return fmt.Errorf("gagal memasukkan management shift untuk karyawan %d: %v", req.IDKaryawan, err)
        }
    }

    // Commit transaksi
    if err = tx.Commit(); err != nil {
        return fmt.Errorf("gagal commit transaksi: %v", err)
    }

    return nil
}