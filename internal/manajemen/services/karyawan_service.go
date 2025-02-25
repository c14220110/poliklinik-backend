package services

import (
	"database/sql"
	"fmt"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
)

func (s *ManagementService) AddKaryawan(karyawan models.Karyawan, role string, idManagement int, createdBy, updatedBy string) (int64, error) {
	// 1. Cek apakah NIK sudah terdaftar
	var idKaryawan int64
	err := s.DB.QueryRow("SELECT id_karyawan FROM Karyawan WHERE NIK = ?", karyawan.NIK).Scan(&idKaryawan)
	if err != sql.ErrNoRows {
		// Jika tidak menghasilkan ErrNoRows, berarti NIK sudah terdaftar atau terjadi error
		return 0, fmt.Errorf("NIK %s sudah terdaftar", karyawan.NIK)
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

	// 4. Insert data karyawan ke tabel Karyawan, termasuk id_role
	insertKaryawan := `
		INSERT INTO Karyawan (Nama, Username, Password, NIK, Tanggal_Lahir, Alamat, No_Telp, id_role)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.DB.Exec(insertKaryawan, karyawan.Nama, karyawan.Username, karyawan.Password, karyawan.NIK, karyawan.TanggalLahir, karyawan.Alamat, karyawan.NoTelp, karyawan.IDRole)
	if err != nil {
		return 0, fmt.Errorf("gagal menambahkan karyawan: %v", err)
	}

	idKaryawan, err = result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("gagal mendapatkan ID Karyawan: %v", err)
	}

	// 5. Insert record di Management_Karyawan untuk mencatat siapa yang menambahkan karyawan
	insertManagement := `
		INSERT INTO Management_Karyawan (id_management, id_karyawan, Created_By, Updated_By)
		VALUES (?, ?, ?, ?)
	`
	_, err = s.DB.Exec(insertManagement, idManagement, idKaryawan, createdBy, updatedBy)
	if err != nil {
		return 0, fmt.Errorf("gagal mencatat di Management_Karyawan: %v", err)
	}

	return idKaryawan, nil
}

func (s *ManagementService) UpdateKaryawan(karyawan models.Karyawan, role string, idManagement int) (int64, error) {
	// Cek apakah ID Karyawan valid
	var idKaryawan int64
	err := s.DB.QueryRow("SELECT ID_Karyawan FROM Karyawan WHERE ID_Karyawan = ?", karyawan.IDKaryawan).Scan(&idKaryawan)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("karyawan dengan ID %d tidak ditemukan", karyawan.IDKaryawan)
	}

	// Update Karyawan
	updateKaryawan := `
		UPDATE Karyawan 
		SET Nama = ?, Username = ?, Password = ?, NIK = ?, Tanggal_Lahir = ?, Alamat = ?, No_Telp = ?
		WHERE ID_Karyawan = ?
	`
	_, err = s.DB.Exec(updateKaryawan, karyawan.Nama, karyawan.Username, karyawan.Password, karyawan.NIK, karyawan.TanggalLahir, karyawan.Alamat, karyawan.NoTelp, karyawan.IDKaryawan)
	if err != nil {
		return 0, fmt.Errorf("gagal mengupdate karyawan: %v", err)
	}

	// Cek apakah Role sudah ada, jika tidak ada, tambah Role baru
	var idRole int64
	err = s.DB.QueryRow("SELECT ID_Role FROM Role WHERE Nama_Role = ?", role).Scan(&idRole)
	if err == sql.ErrNoRows {
		// Insert Role jika tidak ada
		insertRole := "INSERT INTO Role (Nama_Role) VALUES (?)"
		result, err := s.DB.Exec(insertRole, role)
		if err != nil {
			return 0, fmt.Errorf("gagal menambahkan role: %v", err)
		}
		idRole, err = result.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("gagal mendapatkan ID Role: %v", err)
		}
	}

	// Update Detail_Role_Karyawan
	updateDetailRole := `
		UPDATE Detail_Role_Karyawan
		SET ID_Role = ?, Nama = ?, NIK = ?, Alamat = ?, No_Telp = ?
		WHERE ID_Karyawan = ?
	`
	_, err = s.DB.Exec(updateDetailRole, idRole, karyawan.Nama, karyawan.NIK, karyawan.Alamat, karyawan.NoTelp, karyawan.IDKaryawan)
	if err != nil {
		return 0, fmt.Errorf("gagal mengupdate detail role karyawan: %v", err)
	}

	// Update Management_Karyawan untuk mencatat siapa yang mengupdate
	updateManagement := `
		UPDATE Management_Karyawan 
		SET Updated_By = ?
		WHERE ID_Karyawan = ?
	`
	_, err = s.DB.Exec(updateManagement, "admin", karyawan.IDKaryawan)
	if err != nil {
		return 0, fmt.Errorf("gagal mencatat di Management_Karyawan: %v", err)
	}

	return karyawan.IDKaryawan, nil
}

// ManagementService sudah ada, tambahkan fungsi berikut:
func (s *ManagementService) GetKaryawanList() ([]map[string]interface{}, error) {
	query := `
		SELECT 
			k.ID_Karyawan, 
			k.Nama, 
			k.NIK, 
			k.Tanggal_Lahir, 
			r.Nama_Role,
			YEAR(k.Created_At) as Tahun_Kerja
		FROM Karyawan k
		LEFT JOIN Detail_Role_Karyawan drk ON k.ID_Karyawan = drk.ID_Karyawan
		LEFT JOIN Role r ON drk.ID_Role = r.ID_Role
		ORDER BY k.ID_Karyawan
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var idKaryawan int
		var nama, nik string
		var tanggalLahir sql.NullTime
		var role sql.NullString
		var tahunKerja sql.NullInt64

		if err := rows.Scan(&idKaryawan, &nama, &nik, &tanggalLahir, &role, &tahunKerja); err != nil {
			return nil, err
		}

		record := map[string]interface{}{
			"id_karyawan":  idKaryawan,
			"nama":         nama,
			"nik":          nik,
			"tanggal_lahir": nil,
			"role":         nil,
			"tahun_kerja":  nil,
		}
		if tanggalLahir.Valid {
			record["tanggal_lahir"] = tanggalLahir.Time.Format("2006-01-02")
		}
		if role.Valid {
			record["role"] = role.String
		}
		if tahunKerja.Valid {
			record["tahun_kerja"] = tahunKerja.Int64
		}
		list = append(list, record)
	}
	return list, nil
}