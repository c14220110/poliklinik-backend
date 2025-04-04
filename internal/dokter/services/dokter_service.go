package services

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"

	commonModels "github.com/c14220110/poliklinik-backend/internal/common/models"
	"github.com/c14220110/poliklinik-backend/internal/dokter/models"
)

type DokterService struct {
	DB *sql.DB
}

func NewDokterService(db *sql.DB) *DokterService {
	return &DokterService{DB: db}
}

// AuthenticateDokterUsingKaryawan memvalidasi login dokter, mengambil role, privilege, dan shift aktif.
func (s *DokterService) AuthenticateDokterUsingKaryawan(username, password string, selectedPoli int) (*models.Dokter, *commonModels.ShiftKaryawan, error) {
	var dokter models.Dokter
	// Ambil data karyawan dari tabel Karyawan
	queryKaryawan := "SELECT ID_Karyawan, Nama, Username, Password FROM Karyawan WHERE Username = ?"
	err := s.DB.QueryRow(queryKaryawan, username).Scan(&dokter.ID_Dokter, &dokter.Nama, &dokter.Username, &dokter.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.New("username tidak ditemukan")
		}
		return nil, nil, err
	}
	
	// Verifikasi password menggunakan bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(dokter.Password), []byte(password)); err != nil {
		return nil, nil, errors.New("password salah")
	}

	// Ambil role dari Detail_Role_Karyawan dan Role (harus "Dokter")
	var roleName string
	var roleID int
	queryRole := `
		SELECT r.Nama_Role, drk.ID_Role
		FROM Detail_Role_Karyawan drk 
		JOIN Role r ON drk.ID_Role = r.ID_Role
		WHERE drk.ID_Karyawan = ?
		LIMIT 1
	`
	err = s.DB.QueryRow(queryRole, dokter.ID_Dokter).Scan(&roleName, &roleID)
	if err != nil {
		return nil, nil, errors.New("gagal mengambil role")
	}
	if roleName != "Dokter" {
		return nil, nil, errors.New("user bukan Dokter")
	}
	dokter.ID_Role = roleID

	// Ambil daftar privilege
	rows, err := s.DB.Query("SELECT id_privilege FROM Detail_Privilege_Karyawan WHERE id_karyawan = ?", dokter.ID_Dokter)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var privileges []int
	for rows.Next() {
		var priv int
		if err := rows.Scan(&priv); err != nil {
			return nil, nil, err
		}
		privileges = append(privileges, priv)
	}
	dokter.Privileges = privileges

	// Cek shift aktif berdasarkan custom_jam_mulai dan custom_jam_selesai
	var shift commonModels.ShiftKaryawan
	shiftQuery := `
		SELECT sk.ID_Shift_Karyawan, sk.custom_jam_mulai, sk.custom_jam_selesai, sk.ID_Poli
		FROM Shift_Karyawan sk
		WHERE sk.ID_Karyawan = ?
		  AND sk.ID_Poli = ?
		  AND sk.Tanggal = CURDATE()
		  AND (
		     (sk.custom_jam_mulai < sk.custom_jam_selesai AND CURTIME() BETWEEN sk.custom_jam_mulai AND sk.custom_jam_selesai)
		     OR (sk.custom_jam_mulai > sk.custom_jam_selesai AND (CURTIME() >= sk.custom_jam_mulai OR CURTIME() < sk.custom_jam_selesai))
		  )
		LIMIT 1
	`
	err = s.DB.QueryRow(shiftQuery, dokter.ID_Dokter, selectedPoli).
		Scan(&shift.ID_Shift_Karyawan, &shift.CustomJamMulai, &shift.CustomJamSelesai, &shift.ID_Poli)
	if err != nil {
		if err == sql.ErrNoRows {
			// Penanganan untuk kasus tidak ada shift aktif
			return nil, nil, errors.New("tidak ada shift aktif hari ini untuk poli yang dipilih")
		}
		return nil, nil, err
	}
	if shift.ID_Poli != selectedPoli {
		return nil, nil, errors.New("poliklinik yang dipilih tidak sesuai dengan shift aktif")
	}

	return &dokter, &shift, nil
}





// ----------------------------------------------
// Fungsi baru: GetListAntrianByPoli
// Mengembalikan daftar pasien (dengan data dari Pasien, Rekam_Medis, Antrian, dan Poliklinik)
// untuk antrian dengan status = 0 pada poli tertentu.
func (s *DokterService) GetListAntrianByPoli(idPoli int) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			p.Nama, 
			rm.ID_RM, 
			pl.Nama_Poli, 
			a.Nomor_Antrian, 
			a.Status,
			a.ID_Antrian
		FROM Pasien p
		LEFT JOIN Rekam_Medis rm ON p.ID_Pasien = rm.ID_Pasien
		LEFT JOIN Antrian a ON p.ID_Pasien = a.ID_Pasien
		LEFT JOIN Poliklinik pl ON a.ID_Poli = pl.ID_Poli
		WHERE a.ID_Poli = ? AND a.Status = 0
		ORDER BY a.Created_At ASC
	`
	rows, err := s.DB.Query(query, idPoli)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		var nama string
		var idRM sql.NullString
		var namaPoli sql.NullString
		var nomorAntrian sql.NullInt64
		var status int
		var idAntrian int

		if err := rows.Scan(&nama, &idRM, &namaPoli, &nomorAntrian, &status, &idAntrian); err != nil {
			return nil, err
		}

		record := map[string]interface{}{
			"nama":          nama,
			"id_rm":         nil,
			"nama_poli":     nil,
			"nomor_antrian": nil,
			"status":        status,
			"id_billing":    nil, // untuk konsistensi, jika diperlukan; tapi di sini kita ambil id_antrian
			"id_antrian":    idAntrian,
		}

		if idRM.Valid {
			record["id_rm"] = idRM.String
		}
		if namaPoli.Valid {
			record["nama_poli"] = namaPoli.String
		}
		if nomorAntrian.Valid {
			record["nomor_antrian"] = nomorAntrian.Int64
		}

		result = append(result, record)
	}

	return result, nil
}
