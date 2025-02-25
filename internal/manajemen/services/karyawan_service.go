package services

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
	"golang.org/x/crypto/bcrypt"
)


func (s *ManagementService) AddKaryawan(karyawan models.Karyawan, role string, idManagement int, createdBy, updatedBy string) (int64, error) {
	// 1. Cek apakah NIK sudah terdaftar
	var existingID int64
	err := s.DB.QueryRow("SELECT id_karyawan FROM Karyawan WHERE nik = ?", karyawan.NIK).Scan(&existingID)
	if err != sql.ErrNoRows {
		// Jika tidak menghasilkan ErrNoRows (atau err == nil), berarti NIK sudah terdaftar atau terjadi error
		return 0, fmt.Errorf("NIK %s sudah terdaftar", karyawan.NIK)
	}

	// 1b. Cek apakah Username sudah terdaftar
	var existingUsername int64
	err = s.DB.QueryRow("SELECT id_karyawan FROM Karyawan WHERE username = ?", karyawan.Username).Scan(&existingUsername)
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("username %s sudah terdaftar", karyawan.Username)
	}

	// 2. Cek apakah Role sudah ada
	var idRole int64
	err = s.DB.QueryRow("SELECT id_role FROM Role WHERE nama_role = ?", role).Scan(&idRole)
	if err == sql.ErrNoRows {
		// Jika tidak ada, insert role baru
		insertRole := "INSERT INTO Role (nama_role) VALUES (?)"
		result, err := s.DB.Exec(insertRole, role)
		if err != nil {
			return 0, fmt.Errorf("gagal menambahkan role: %v", err)
		}
		idRole, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("gagal mendapatkan ID Role: %v", err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("gagal memeriksa role: %v", err)
	}

	// 3. Set id_role pada objek karyawan
	karyawan.IDRole = idRole

	// 4. Hash password sebelum disimpan
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(karyawan.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %v", err)
	}
	karyawan.Password = string(hashedPassword)

	// 5. Insert data karyawan ke tabel Karyawan, termasuk id_role
	insertKaryawan := `
		INSERT INTO Karyawan (nama, username, password, nik, tanggal_lahir, alamat, no_telp, id_role)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.DB.Exec(insertKaryawan,
		karyawan.Nama,
		karyawan.Username,
		karyawan.Password,
		karyawan.NIK,
		karyawan.TanggalLahir,
		karyawan.Alamat,
		karyawan.NoTelp,
		karyawan.IDRole,
	)
	if err != nil {
		return 0, fmt.Errorf("gagal menambahkan karyawan: %v", err)
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("gagal mendapatkan ID Karyawan: %v", err)
	}

	// 6. Insert record di Management_Karyawan untuk mencatat siapa yang menambahkan karyawan
	insertManagement := `
		INSERT INTO Management_Karyawan (id_management, id_karyawan, Created_By, Updated_By)
		VALUES (?, ?, ?, ?)
	`
	_, err = s.DB.Exec(insertManagement, idManagement, newID, createdBy, updatedBy)
	if err != nil {
		return 0, fmt.Errorf("gagal mencatat di Management_Karyawan: %v", err)
	}

	return newID, nil
}


func (s *ManagementService) UpdateKaryawan(karyawan models.Karyawan, role string, idManagement int, updatedBy string) (int64, error) {
	// 1. Cek apakah ID_Karyawan valid
	var idKaryawan int64
	err := s.DB.QueryRow("SELECT id_karyawan FROM Karyawan WHERE id_karyawan = ?", karyawan.IDKaryawan).Scan(&idKaryawan)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("karyawan dengan ID %d tidak ditemukan", karyawan.IDKaryawan)
	} else if err != nil {
		return 0, fmt.Errorf("error checking karyawan: %v", err)
	}

	// 2. Cek duplikasi username (selain record ini)
	var count int
	err = s.DB.QueryRow("SELECT COUNT(*) FROM Karyawan WHERE username = ? AND id_karyawan <> ?", karyawan.Username, karyawan.IDKaryawan).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("gagal memeriksa duplikasi username: %v", err)
	}
	if count > 0 {
		return 0, fmt.Errorf("username %s sudah digunakan", karyawan.Username)
	}

	// 3. Cek duplikasi NIK (selain record ini)
	count = 0
	err = s.DB.QueryRow("SELECT COUNT(*) FROM Karyawan WHERE nik = ? AND id_karyawan <> ?", karyawan.NIK, karyawan.IDKaryawan).Scan(&count)
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

	// 5. Cari atau buat Role yang sesuai
	var idRole int64
	err = s.DB.QueryRow("SELECT id_role FROM Role WHERE nama_role = ?", role).Scan(&idRole)
	if err == sql.ErrNoRows {
		// Insert Role jika tidak ada
		insertRole := "INSERT INTO Role (nama_role) VALUES (?)"
		result, err := s.DB.Exec(insertRole, role)
		if err != nil {
			return 0, fmt.Errorf("gagal menambahkan role: %v", err)
		}
		idRole, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("gagal mendapatkan ID Role: %v", err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("gagal memeriksa role: %v", err)
	}

	// 6. Update record Karyawan termasuk kolom id_role
	updateKaryawan := `
		UPDATE Karyawan 
		SET nama = ?, username = ?, password = ?, nik = ?, tanggal_lahir = ?, alamat = ?, no_telp = ?, id_role = ?
		WHERE id_karyawan = ?
	`
	_, err = s.DB.Exec(updateKaryawan,
		karyawan.Nama,
		karyawan.Username,
		karyawan.Password,
		karyawan.NIK,
		karyawan.TanggalLahir,
		karyawan.Alamat,
		karyawan.NoTelp,
		idRole,
		karyawan.IDKaryawan,
	)
	if err != nil {
		return 0, fmt.Errorf("gagal mengupdate karyawan: %v", err)
	}

	// 7. Update Management_Karyawan untuk mencatat siapa yang melakukan pembaruan
	updateManagement := `
		UPDATE Management_Karyawan 
		SET Updated_By = ?
		WHERE id_karyawan = ?
	`
	_, err = s.DB.Exec(updateManagement, updatedBy, karyawan.IDKaryawan)
	if err != nil {
		return 0, fmt.Errorf("gagal mencatat di Management_Karyawan: %v", err)
	}

	return karyawan.IDKaryawan, nil
}

func (s *ManagementService) GetKaryawanListFiltered(idRoleFilter string, statusFilter string) ([]map[string]interface{}, error) {
	baseQuery := `
		SELECT 
			k.id_karyawan, 
			k.id_role,
			r.nama_role,
			k.nama,
			k.username,
			k.nik,
			k.tanggal_lahir,
			k.alamat,
			k.no_telp
		FROM Karyawan k
		JOIN Role r ON k.id_role = r.id_role
	`
	conditions := []string{}
	params := []interface{}{}

	// Filter berdasarkan id_role jika disediakan
	if idRoleFilter != "" {
		idRoleInt, err := strconv.Atoi(idRoleFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid id_role value: %v", err)
		}
		conditions = append(conditions, "k.id_role = ?")
		params = append(params, idRoleInt)
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
	query += " ORDER BY k.id_karyawan"

	rows, err := s.DB.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var idKaryawan int
		var idRole int
		var namaRole sql.NullString
		var nama, username, nik string
		var tanggalLahir sql.NullTime
		var alamat, noTelp string

		if err := rows.Scan(&idKaryawan, &idRole, &namaRole, &nama, &username, &nik, &tanggalLahir, &alamat, &noTelp); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}

		record := map[string]interface{}{
			"id_karyawan":  idKaryawan,
			"id_role":      idRole,
			"nama_role":    nil,
			"nama":         nama,
			"username":     username,
			"nik":          nik,
			"tanggal_lahir": nil,
			"alamat":       alamat,
			"no_telp":      noTelp,
		}
		if namaRole.Valid {
			record["nama_role"] = namaRole.String
		}
		if tanggalLahir.Valid {
			record["tanggal_lahir"] = tanggalLahir.Time.Format("2006-01-02")
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