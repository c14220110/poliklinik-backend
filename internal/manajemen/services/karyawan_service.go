package services

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
	"golang.org/x/crypto/bcrypt"
)


func (s *ManagementService) AddKaryawan(karyawan models.Karyawan, roles []string, idManagement, createdBy, updatedBy int) (int64, error) {
	// Mulai transaksi
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}

	// Pastikan rollback jika terjadi error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 1. Cek apakah NIK sudah terdaftar
	var existingID int64
	err = tx.QueryRow("SELECT id_karyawan FROM Karyawan WHERE nik = ?", karyawan.NIK).Scan(&existingID)
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("NIK %s sudah terdaftar", karyawan.NIK)
	}

	// 1b. Cek apakah Username sudah terdaftar
	var existingUsername int64
	err = tx.QueryRow("SELECT id_karyawan FROM Karyawan WHERE username = ?", karyawan.Username).Scan(&existingUsername)
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("username %s sudah terdaftar", karyawan.Username)
	}

	// 1c. Cek apakah SIP sudah terdaftar (jika disediakan)
	if karyawan.Sip != "" {
		var existingSipID int64
		err = tx.QueryRow("SELECT id_karyawan FROM Karyawan WHERE sip = ?", karyawan.Sip).Scan(&existingSipID)
		if err != sql.ErrNoRows {
			return 0, fmt.Errorf("SIP %s sudah terdaftar", karyawan.Sip)
		}
	}

	// 2. Cek dan tambahkan role jika belum ada, lalu kumpulkan id_role
	roleIDs := make([]int64, 0, len(roles))
	for _, role := range roles {
		var idRole int64
		err = tx.QueryRow("SELECT id_role FROM Role WHERE nama_role = ?", role).Scan(&idRole)
		if err == sql.ErrNoRows {
			// Jika role tidak ada, insert role baru
			insertRole := "INSERT INTO Role (nama_role) VALUES (?)"
			res, err := tx.Exec(insertRole, role)
			if err != nil {
				return 0, fmt.Errorf("gagal menambahkan role %s: %v", role, err)
			}
			idRole, err = res.LastInsertId()
			if err != nil {
				return 0, fmt.Errorf("gagal mendapatkan ID Role untuk %s: %v", role, err)
			}
		} else if err != nil {
			return 0, fmt.Errorf("gagal memeriksa role %s: %v", role, err)
		}
		roleIDs = append(roleIDs, idRole)
	}

	// 4. Hash password sebelum disimpan
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(karyawan.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %v", err)
	}
	karyawan.Password = string(hashedPassword)

	// 5. Insert data karyawan ke tabel Karyawan
	insertKaryawan := `
		INSERT INTO Karyawan (nama, username, password, nik, tanggal_lahir, alamat, no_telp, jenis_kelamin, sip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	var sipValue interface{}
	if karyawan.Sip != "" {
		sipValue = karyawan.Sip
	} else {
		sipValue = nil
	}
	res, err := tx.Exec(insertKaryawan,
		karyawan.Nama,
		karyawan.Username,
		karyawan.Password,
		karyawan.NIK,
		karyawan.TanggalLahir,
		karyawan.Alamat,
		karyawan.NoTelp,
		karyawan.JenisKelamin,
		sipValue,
	)
	if err != nil {
		return 0, fmt.Errorf("gagal menambahkan karyawan: %v", err)
	}

	newID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("gagal mendapatkan ID Karyawan: %v", err)
	}

	// 6. Insert record di Management_Karyawan
	insertManagement := `
		INSERT INTO Management_Karyawan (id_management, id_karyawan, created_by, updated_by, deleted_by)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(insertManagement, idManagement, newID, createdBy, updatedBy, nil)
	if err != nil {
		return 0, fmt.Errorf("gagal mencatat di Management_Karyawan: %v", err)
	}

	// 7. Insert record di Detail_Role_Karyawan untuk setiap role
	insertDetailRole := `
		INSERT INTO Detail_Role_Karyawan (id_role, id_karyawan)
		VALUES (?, ?)
	`
	for _, idRole := range roleIDs {
		_, err = tx.Exec(insertDetailRole, idRole, newID)
		if err != nil {
			return 0, fmt.Errorf("gagal mencatat role ID %d di Detail_Role_Karyawan: %v", idRole, err)
		}
	}

	// Commit transaksi
	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return newID, nil
}


func (s *ManagementService) UpdateKaryawan(karyawan models.Karyawan, role string, idManagement int) (int64, error) {
	// Mulai transaksi
	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}

	// Pastikan rollback jika terjadi error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 1. Cek apakah karyawan dengan ID tersebut ada
	var existingID int64
	err = tx.QueryRow("SELECT id_karyawan FROM Karyawan WHERE id_karyawan = ?", karyawan.IDKaryawan).Scan(&existingID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("karyawan dengan ID %d tidak ditemukan", karyawan.IDKaryawan)
	} else if err != nil {
		return 0, fmt.Errorf("error checking karyawan: %v", err)
	}

	// 2. Cek duplikasi username (selain record ini)
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM Karyawan WHERE username = ? AND id_karyawan <> ?", karyawan.Username, karyawan.IDKaryawan).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("gagal memeriksa duplikasi username: %v", err)
	}
	if count > 0 {
		return 0, fmt.Errorf("username %s sudah digunakan", karyawan.Username)
	}

	// 3. Cek duplikasi NIK (selain record ini)
	count = 0
	err = tx.QueryRow("SELECT COUNT(*) FROM Karyawan WHERE nik = ? AND id_karyawan <> ?", karyawan.NIK, karyawan.IDKaryawan).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("gagal memeriksa duplikasi NIK: %v", err)
	}
	if count > 0 {
		return 0, fmt.Errorf("NIK %s sudah digunakan", karyawan.NIK)
	}

	// 4. Hash password baru (jika diupdate)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(karyawan.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %v", err)
	}
	karyawan.Password = string(hashedPassword)

	// 5. Cari Role yang sesuai (role harus sudah ada, tidak membuat role baru)
	var idRole int64
	err = tx.QueryRow("SELECT id_role FROM Role WHERE nama_role = ?", role).Scan(&idRole)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("role %s tidak ditemukan", role)
	} else if err != nil {
		return 0, fmt.Errorf("gagal memeriksa role: %v", err)
	}

	// 6. Update record Karyawan (tidak ada kolom id_role)
	updateKaryawan := `
		UPDATE Karyawan 
		SET nama = ?, username = ?, password = ?, nik = ?, tanggal_lahir = ?, alamat = ?, no_telp = ?
		WHERE id_karyawan = ?
	`
	_, err = tx.Exec(updateKaryawan,
		karyawan.Nama,
		karyawan.Username,
		karyawan.Password,
		karyawan.NIK,
		karyawan.TanggalLahir,
		karyawan.Alamat,
		karyawan.NoTelp,
		karyawan.IDKaryawan,
	)
	if err != nil {
		return 0, fmt.Errorf("gagal mengupdate karyawan: %v", err)
	}

	// 7. Update Management_Karyawan untuk mencatat siapa yang melakukan pembaruan
	updateManagement := `
		UPDATE Management_Karyawan 
		SET updated_by = ?
		WHERE id_karyawan = ?
	`
	_, err = tx.Exec(updateManagement, idManagement, karyawan.IDKaryawan)
	if err != nil {
		return 0, fmt.Errorf("gagal mencatat di Management_Karyawan: %v", err)
	}

	// 8. Update Detail_Role_Karyawan:
	// Hapus record role lama dan masukkan record baru untuk karyawan tersebut
	_, err = tx.Exec("DELETE FROM Detail_Role_Karyawan WHERE id_karyawan = ?", karyawan.IDKaryawan)
	if err != nil {
		return 0, fmt.Errorf("gagal menghapus detail role lama: %v", err)
	}

	insertDetailRole := `
		INSERT INTO Detail_Role_Karyawan (id_role, id_karyawan)
		VALUES (?, ?)
	`
	_, err = tx.Exec(insertDetailRole, idRole, karyawan.IDKaryawan)
	if err != nil {
		return 0, fmt.Errorf("gagal mencatat detail role baru: %v", err)
	}

	// Commit transaksi
	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return karyawan.IDKaryawan, nil
}

func (s *ManagementService) GetKaryawanListFiltered(namaRoleFilter string, statusFilter string, idKaryawanFilter string) ([]map[string]interface{}, error) {
	// Base query dengan GROUP BY untuk mengelompokkan per karyawan dan menggabungkan role
	baseQuery := `
		SELECT 
			k.id_karyawan, 
			GROUP_CONCAT(r.nama_role SEPARATOR ', ') AS roles,
			k.nama,
			k.username,
			k.nik,
			k.tanggal_lahir,
			k.alamat,
			k.no_telp,
			k.sip AS nomor_sip
		FROM Karyawan k
		LEFT JOIN Detail_Role_Karyawan drk ON k.id_karyawan = drk.id_karyawan
		LEFT JOIN Role r ON drk.id_role = r.id_role
	`
	conditions := []string{}
	params := []interface{}{}

	// Filter berdasarkan id_karyawan jika disediakan
	if idKaryawanFilter != "" {
		idKaryawanInt, err := strconv.Atoi(idKaryawanFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid id_karyawan value: %v", err)
		}
		conditions = append(conditions, "k.id_karyawan = ?")
		params = append(params, idKaryawanInt)
	}

	// Filter berdasarkan nama_role jika disediakan
	if namaRoleFilter != "" {
		roleNames := strings.Split(namaRoleFilter, ",")
		roleConditions := []string{}
		for _, role := range roleNames {
			roleConditions = append(roleConditions, "r.nama_role = ?")
			params = append(params, strings.TrimSpace(role))
		}
		// Menggunakan OR untuk mencari karyawan yang memiliki setidaknya salah satu role
		conditions = append(conditions, "("+strings.Join(roleConditions, " OR ")+")")
	}

	// Filter berdasarkan status
	if statusFilter != "" {
		statusLower := strings.ToLower(statusFilter)
		if statusLower == "aktif" {
			conditions = append(conditions, "k.deleted_at IS NULL")
		} else if statusLower == "nonaktif" {
			conditions = append(conditions, "k.deleted_at IS NOT NULL")
		}
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " GROUP BY k.id_karyawan"
	query += " ORDER BY k.id_karyawan"

	rows, err := s.DB.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var idKaryawan int
		var roles sql.NullString
		var nama, username, nik string
		var tanggalLahir sql.NullTime
		var alamat, noTelp string
		var nomorSip sql.NullString

		if err := rows.Scan(&idKaryawan, &roles, &nama, &username, &nik, &tanggalLahir, &alamat, &noTelp, &nomorSip); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}

		record := map[string]interface{}{
			"id_karyawan":   idKaryawan,
			"roles":         nil,
			"nama":          nama,
			"username":      username,
			"nik":           nik,
			"tanggal_lahir": nil,
			"alamat":        alamat,
			"no_telp":       noTelp,
		}
		if roles.Valid {
			record["roles"] = strings.Split(roles.String, ", ")
		}
		if tanggalLahir.Valid {
			record["tanggal_lahir"] = tanggalLahir.Time.Format("2006-01-02")
		}
		if nomorSip.Valid {
			record["nomor_sip"] = nomorSip.String
		}
		list = append(list, record)
	}
	return list, nil
}

func (s *ManagementService) SoftDeleteKaryawan(idKaryawan int, deletedBy string) error {
	// 1. Update kolom deleted_at di tabel Karyawan
	queryKaryawan := `UPDATE Karyawan SET deleted_at = NOW() WHERE id_karyawan = ?`
	_, err := s.DB.Exec(queryKaryawan, idKaryawan)
	if err != nil {
		return fmt.Errorf("failed to soft delete karyawan: %v", err)
	}

	// 2. Update kolom deleted_by di tabel Management_Karyawan
	queryManagement := `UPDATE Management_Karyawan SET deleted_by = ? WHERE id_karyawan = ?`
	_, err = s.DB.Exec(queryManagement, deletedBy, idKaryawan)
	if err != nil {
		return fmt.Errorf("failed to update Management_Karyawan: %v", err)
	}

	return nil
}

func (s *ManagementService) AddPrivilegesToKaryawan(idKaryawan int, privileges []int) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	// Pastikan rollback jika terjadi error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	
	// Loop untuk setiap privilege yang akan ditambahkan
	for _, priv := range privileges {
		// Opsional: cek apakah privilege sudah ada untuk mencegah duplikasi
		var exists int
		err = tx.QueryRow("SELECT COUNT(*) FROM Detail_Privilege_Karyawan WHERE id_privilege = ? AND id_karyawan = ?", priv, idKaryawan).Scan(&exists)
		if err != nil {
			return fmt.Errorf("gagal memeriksa privilege: %v", err)
		}
		if exists > 0 {
			// Lewati jika privilege sudah terdaftar (atau bisa mengembalikan error sesuai kebutuhan)
			continue
		}
		_, err = tx.Exec("INSERT INTO Detail_Privilege_Karyawan (id_privilege, id_karyawan) VALUES (?, ?)", priv, idKaryawan)
		if err != nil {
			return fmt.Errorf("gagal menambahkan privilege %d untuk karyawan %d: %v", priv, idKaryawan, err)
		}
	}
	
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}
