package services

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
)

type ShiftService struct {
	DB *sql.DB
}

func NewShiftService(db *sql.DB) *ShiftService {
	return &ShiftService{DB: db}
}


func (s *ShiftService) AssignShiftKaryawan(
	idPoli, idShift int,
	tanggal string,
	idManagement int,
	items []models.ShiftAssignItem,
) ([]int64, error) {

	tx, err := s.DB.Begin()
	if err != nil { return nil, err }

	// pastikan id_shift ada
	var dummy int
	if err := tx.QueryRow("SELECT 1 FROM Shift WHERE id_shift = ?", idShift).Scan(&dummy); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows { return nil, fmt.Errorf("id_shift %d tidak ditemukan", idShift) }
		return nil, err
	}

	var insertedIDs []int64

	for _, it := range items {
		// validasi jam
		if _, err := time.Parse("15:04:05", it.JamMulai); err != nil {
			tx.Rollback(); return nil, fmt.Errorf("jam_mulai '%s' tidak valid", it.JamMulai)
		}
		if _, err := time.Parse("15:04:05", it.JamAkhir); err != nil {
			tx.Rollback(); return nil, fmt.Errorf("jam_akhir '%s' tidak valid", it.JamAkhir)
		}

		// loop setiap role
		for _, roleName := range it.NamaRole {
			var idRole int
			if err := tx.QueryRow("SELECT id_role FROM Role WHERE nama_role = ?", roleName).
				Scan(&idRole); err != nil {
				tx.Rollback()
				if err == sql.ErrNoRows {
					return nil, fmt.Errorf("role '%s' tidak ditemukan", roleName)
				}
				return nil, err
			}

			// cek karyawan punya role tsb
			var cnt int
			if err := tx.QueryRow(
				"SELECT COUNT(*) FROM Detail_Role_Karyawan WHERE id_karyawan=? AND id_role=?",
				it.IDKaryawan, idRole).Scan(&cnt); err != nil {
				tx.Rollback(); return nil, err
			}
			if cnt == 0 {
				tx.Rollback()
				return nil, fmt.Errorf("karyawan %d tidak memiliki role '%s'", it.IDKaryawan, roleName)
			}

			// cek duplikat shift
			if err := tx.QueryRow(
				`SELECT COUNT(*) FROM Shift_Karyawan
				  WHERE id_karyawan=? AND id_poli=? AND id_shift=? AND id_role=? AND tanggal=?`,
				it.IDKaryawan, idPoli, idShift, idRole, tanggal).Scan(&cnt); err != nil {
				tx.Rollback(); return nil, err
			}
			if cnt > 0 {
				tx.Rollback()
				return nil, fmt.Errorf("karyawan %d dengan role '%s' sudah punya shift pada tanggal %s",
					it.IDKaryawan, roleName, tanggal)
			}

			// insert Shift_Karyawan
			res, err := tx.Exec(
				`INSERT INTO Shift_Karyawan
				   (id_poli,id_shift,id_karyawan,id_role,custom_jam_mulai,custom_jam_selesai,tanggal)
				 VALUES (?,?,?,?,?,?,?)`,
				idPoli, idShift, it.IDKaryawan, idRole, it.JamMulai, it.JamAkhir, tanggal)
			if err != nil { tx.Rollback(); return nil, err }

			idShiftKaryawan, _ := res.LastInsertId()
			insertedIDs = append(insertedIDs, idShiftKaryawan)

			deletedBy := sql.NullInt64{Valid: false} // Default ke NULL
			if _, err := tx.Exec(
					`INSERT INTO Management_Shift_Karyawan (id_management, id_shift_karyawan, created_by, updated_by, deleted_by) VALUES (?, ?, ?, ?, ?)`,
					idManagement, idShiftKaryawan, idManagement, idManagement, deletedBy); err != nil {
					tx.Rollback(); return nil, err
			}

		}
	}

	if err := tx.Commit(); err != nil { return nil, err }
	return insertedIDs, nil
}


func (s *ShiftService) UpdateCustomShift(idShiftKaryawan int, newCustomMulai, newCustomSelesai string) error {
	// Mulai transaksi
	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi: %v", err)
	}

	// Parse waktu custom yang baru dengan format "15:04:05"
	newMulai, err := time.Parse("15:04:05", newCustomMulai)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("format custom_jam_mulai tidak valid: %v", err)
	}
	newSelesai, err := time.Parse("15:04:05", newCustomSelesai)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("format custom_jam_selesai tidak valid: %v", err)
	}

	// Ambil data dari Shift_Karyawan dan Shift untuk validasi
	var (
		shiftJamMulai, shiftJamSelesai string
	)
	query := `
		SELECT s.jam_mulai, s.jam_selesai 
		FROM Shift_Karyawan sk
		JOIN Shift s ON sk.id_shift = s.id_shift
		WHERE sk.id_shift_karyawan = ?
	`
	err = tx.QueryRow(query, idShiftKaryawan).Scan(&shiftJamMulai, &shiftJamSelesai)
	if err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return fmt.Errorf("record Shift_Karyawan dengan id %d tidak ditemukan", idShiftKaryawan)
		}
		return fmt.Errorf("gagal mengambil data shift: %v", err)
	}

	// Parse default waktu shift
	defaultMulai, err := time.Parse("15:04:05", shiftJamMulai)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("format default jam_mulai tidak valid: %v", err)
	}
	defaultSelesai, err := time.Parse("15:04:05", shiftJamSelesai)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("format default jam_selesai tidak valid: %v", err)
	}

	// Validasi: custom_jam_mulai tidak boleh sebelum default dan custom_jam_selesai tidak boleh melewati default
	if newMulai.Before(defaultMulai) || newSelesai.After(defaultSelesai) {
		tx.Rollback()
		return fmt.Errorf("custom shift harus berada dalam rentang waktu %s - %s", shiftJamMulai, shiftJamSelesai)
	}

	// Update record Shift_Karyawan dengan waktu custom yang baru
	updateQuery := `
		UPDATE Shift_Karyawan 
		SET custom_jam_mulai = ?, custom_jam_selesai = ?
		WHERE id_shift_karyawan = ?
	`
	res, err := tx.Exec(updateQuery, newCustomMulai, newCustomSelesai, idShiftKaryawan)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("gagal update shift: %v", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("gagal memeriksa jumlah record yang diupdate: %v", err)
	}
	if affected == 0 {
		tx.Rollback()
		return fmt.Errorf("tidak ada record yang diupdate")
	}

	// Commit transaksi
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit transaksi: %v", err)
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

	// Gunakan query utama dengan GROUP_CONCAT untuk semua role
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
			GROUP_CONCAT(DISTINCT r.nama_role SEPARATOR ', ') AS roles
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

	// Tambahkan filter id_role jika disediakan
	if strings.TrimSpace(idRoleFilter) != "" {
		idRole, err := strconv.Atoi(idRoleFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid id_role value: %v", err)
		}
		var roleName string
		switch idRole {
		case 1:
			roleName = "Administrasi"
		case 2:
			roleName = "Suster"
		case 3:
			roleName = "Dokter"
		default:
			return nil, fmt.Errorf("id_role %d tidak valid", idRole)
		}
		query += " HAVING GROUP_CONCAT(DISTINCT r.nama_role SEPARATOR ', ') LIKE ?"
		args = append(args, fmt.Sprintf("%%%s%%", roleName))
	}

	// Eksekusi query
	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	// Simpan hasil sementara
	var tempResults []struct {
		idKaryawan       int
		nama             string
		nik              string
		username         string
		noTelp           string
		alamat           string
		jenisKelamin     string
		customJamMulai   string
		customJamSelesai string
		idShiftKaryawan  int
		rolesStr         string
	}
	for rows.Next() {
		var temp struct {
			idKaryawan       int
			nama             string
			nik              string
			username         string
			noTelp           string
			alamat           string
			jenisKelamin     string
			customJamMulai   string
			customJamSelesai string
			idShiftKaryawan  int
			rolesStr         sql.NullString
		}
		if err := rows.Scan(
			&temp.idKaryawan, &temp.nama, &temp.nik, &temp.username, &temp.noTelp,
			&temp.alamat, &temp.jenisKelamin, &temp.customJamMulai,
			&temp.customJamSelesai, &temp.idShiftKaryawan, &temp.rolesStr,
		); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}
		var rolesStr string
		if temp.rolesStr.Valid {
			rolesStr = temp.rolesStr.String
		}
		tempResults = append(tempResults, struct {
			idKaryawan       int
			nama             string
			nik              string
			username         string
			noTelp           string
			alamat           string
			jenisKelamin     string
			customJamMulai   string
			customJamSelesai string
			idShiftKaryawan  int
			rolesStr         string
		}{temp.idKaryawan, temp.nama, temp.nik, temp.username, temp.noTelp, temp.alamat, temp.jenisKelamin, temp.customJamMulai, temp.customJamSelesai, temp.idShiftKaryawan, rolesStr})
	}

	// Proses pengelompokan role
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
				"id_karyawan":        tr.idKaryawan,
				"nama":               tr.nama,
				"NIK":                tr.nik,
				"username":           tr.username,
				"roles":              strings.Join(otherRoles, ", "),
				"no_telp":            tr.noTelp,
				"alamat":             tr.alamat,
				"jenis_kelamin":      tr.jenisKelamin,
				"custom_jam_mulai":   tr.customJamMulai,
				"custom_jam_selesai": tr.customJamSelesai,
				"id_shift_karyawan":  tr.idShiftKaryawan,
			}
			results = append(results, record)
		}
		// Tambahkan record terpisah untuk Dokter jika ada
		if hasDokter {
			record := map[string]interface{}{
				"id_karyawan":        tr.idKaryawan,
				"nama":               tr.nama,
				"NIK":                tr.nik,
				"username":           tr.username,
				"roles":              "Dokter",
				"no_telp":            tr.noTelp,
				"alamat":             tr.alamat,
				"jenis_kelamin":      tr.jenisKelamin,
				"custom_jam_mulai":   tr.customJamMulai,
				"custom_jam_selesai": tr.customJamSelesai,
				"id_shift_karyawan":  tr.idShiftKaryawan,
			}
			results = append(results, record)
		}
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

// AssignShiftRequest adalah struktur untuk request assign shift
type AssignShiftRequest struct {
	IDKaryawan int      `json:"id_karyawan"`
	NamaRole   []string `json:"nama_role"`
	JamMulai   string   `json:"jam_mulai"`
	JamAkhir   string   `json:"jam_akhir"`
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

		// 5. Validasi dan konversi nama_role ke id_role
		if len(req.NamaRole) == 0 {
			tx.Rollback()
			return fmt.Errorf("nama_role untuk karyawan %d tidak boleh kosong", req.IDKaryawan)
		}

		roleMap := map[string]int{
			"Administrasi": 1,
			"Suster":       2,
			"Dokter":       3,
		}

		// Track role yang sudah diproses untuk karyawan ini dalam transaksi
		processedRoles := make(map[int]bool)

		for _, roleName := range req.NamaRole {
			idRole, ok := roleMap[roleName]
			if !ok {
				tx.Rollback()
				return fmt.Errorf("nama_role %s untuk karyawan %d tidak valid", roleName, req.IDKaryawan)
			}

			// Cek apakah role sudah diproses untuk karyawan ini dalam transaksi
			if processedRoles[idRole] {
				tx.Rollback()
				return fmt.Errorf("role %s untuk karyawan %d duplikat dalam request", roleName, req.IDKaryawan)
			}
			processedRoles[idRole] = true

			// 6. Cek apakah karyawan memiliki role yang sesuai
			var roleCount int
			err = tx.QueryRow(
				"SELECT COUNT(*) FROM Detail_Role_Karyawan WHERE id_karyawan = ? AND id_role = ?",
				req.IDKaryawan, idRole,
			).Scan(&roleCount)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("gagal memeriksa role %s untuk karyawan %d: %v", roleName, req.IDKaryawan, err)
			}
			if roleCount == 0 {
				tx.Rollback()
				return fmt.Errorf("karyawan dengan id %d tidak memiliki role %s", req.IDKaryawan, roleName)
			}

			// 7. Cek apakah karyawan sudah memiliki shift dengan role yang sama
			var existingCount int
			err = tx.QueryRow(
				"SELECT COUNT(*) FROM Shift_Karyawan WHERE id_karyawan = ? AND id_poli = ? AND id_shift = ? AND tanggal = ? AND id_role = ?",
				req.IDKaryawan, idPoli, idShift, tanggalFormatted, idRole,
			).Scan(&existingCount)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("gagal memeriksa shift yang ada untuk karyawan %d dengan role %s: %v", req.IDKaryawan, roleName, err)
			}
			if existingCount > 0 {
				tx.Rollback()
				return fmt.Errorf("karyawan %d sudah memiliki shift %d di poli %d pada tanggal %s dengan role %s", req.IDKaryawan, idShift, idPoli, tanggalStr, roleName)
			}

			// 8. Insert ke tabel Shift_Karyawan untuk setiap role
			insertQuery := `
				INSERT INTO Shift_Karyawan (id_poli, id_shift, id_karyawan, id_role, custom_jam_mulai, custom_jam_selesai, tanggal)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`
			res, err := tx.Exec(insertQuery, idPoli, idShift, req.IDKaryawan, idRole, req.JamMulai, req.JamAkhir, tanggalFormatted)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("gagal memasukkan shift untuk karyawan %d dengan role %s: %v", req.IDKaryawan, roleName, err)
			}
			idShiftKaryawan, err := res.LastInsertId()
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("gagal mendapatkan id_shift_karyawan untuk karyawan %d dengan role %s: %v", req.IDKaryawan, roleName, err)
			}

			// 9. Insert ke tabel Management_Shift_Karyawan
			insertManagementShiftQuery := `
				INSERT INTO Management_Shift_Karyawan (id_management, id_shift_karyawan, created_by, updated_by, deleted_by)
				VALUES (?, ?, ?, ?, ?)
			`
			_, err = tx.Exec(insertManagementShiftQuery, idManagement, idShiftKaryawan, idManagement, idManagement, nil)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("gagal memasukkan management shift untuk karyawan %d dengan role %s: %v", req.IDKaryawan, roleName, err)
			}
		}
	}

	// Commit transaksi
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit transaksi: %v", err)
	}

	return nil
}