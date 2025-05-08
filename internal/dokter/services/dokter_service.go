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


// AuthenticateDokterUsingKaryawan memvalidasi login.
// ID_Karyawan == 69 = super‑user: bypass role & shift.
func (s *DokterService) AuthenticateDokterUsingKaryawan(
	username, password string,
	selectedPoli int,
) (*models.Dokter, *commonModels.ShiftKaryawan, error) {

	/* ---------- 1. Ambil data karyawan ---------- */
	var dokter models.Dokter
	if err := s.DB.QueryRow(`
		SELECT ID_Karyawan, Nama, Username, Password
		FROM Karyawan
		WHERE Username = ?`,
		username,
	).Scan(&dokter.ID_Dokter, &dokter.Nama,
		&dokter.Username, &dokter.Password); err != nil {

		if err == sql.ErrNoRows {
			return nil, nil, errors.New("username tidak ditemukan")
		}
		return nil, nil, err
	}

	// Verifikasi password
	if err := bcrypt.CompareHashAndPassword(
		[]byte(dokter.Password), []byte(password)); err != nil {
		return nil, nil, errors.New("password salah")
	}

	/* ---------- 2. Super‑user bypass ---------- */
	if dokter.ID_Dokter == 40 {
		// ambil satu role apa saja (optional)
		_ = s.DB.QueryRow(`
			SELECT IFNULL(ID_Role,0) FROM Detail_Role_Karyawan
			WHERE ID_Karyawan = ? LIMIT 1`, dokter.ID_Dokter).
			Scan(&dokter.ID_Role)

		dummy := &commonModels.ShiftKaryawan{
			ID_Shift_Karyawan: 0,
			CustomJamMulai:    "00:00:00",
			CustomJamSelesai:  "23:59:59",
			ID_Poli:           selectedPoli,
		}
		return &dokter, dummy, nil
	}

	/* ---------- 3. Role harus 'Dokter' ---------- */
	var roleName string
	if err := s.DB.QueryRow(`
		SELECT r.Nama_Role, drk.ID_Role
		FROM Detail_Role_Karyawan drk
		JOIN Role r ON drk.ID_Role = r.ID_Role
		WHERE drk.ID_Karyawan = ?
		LIMIT 1`,
		dokter.ID_Dokter,
	).Scan(&roleName, &dokter.ID_Role); err != nil {

		return nil, nil, errors.New("gagal mengambil role")
	}
	if roleName != "Dokter" {
		return nil, nil, errors.New("user bukan Dokter")
	}

	/* ---------- 4. Privilege ---------- */
	rows, err := s.DB.Query(`
		SELECT id_privilege
		FROM Detail_Privilege_Karyawan
		WHERE id_karyawan = ?`,
		dokter.ID_Dokter)
	if err != nil { return nil, nil, err }
	defer rows.Close()
	for rows.Next() {
		var p int
		if err := rows.Scan(&p); err != nil { return nil, nil, err }
		dokter.Privileges = append(dokter.Privileges, p)
	}

	/* ---------- 5. Shift aktif ---------- */
	var shift commonModels.ShiftKaryawan
	if err := s.DB.QueryRow(`
		SELECT ID_Shift_Karyawan, custom_jam_mulai, custom_jam_selesai, ID_Poli
		FROM Shift_Karyawan
		WHERE ID_Karyawan = ?
		  AND ID_Poli     = ?
		  AND Tanggal     = CURDATE()
		  AND (
		       (custom_jam_mulai <  custom_jam_selesai
		            AND CURTIME() BETWEEN custom_jam_mulai AND custom_jam_selesai)
		    OR (custom_jam_mulai >= custom_jam_selesai
		            AND (CURTIME() >= custom_jam_mulai OR CURTIME() < custom_jam_selesai))
		  )
		LIMIT 1`,
		dokter.ID_Dokter, selectedPoli).
		Scan(&shift.ID_Shift_Karyawan, &shift.CustomJamMulai,
		     &shift.CustomJamSelesai, &shift.ID_Poli); err != nil {

		if err == sql.ErrNoRows {
			return nil, nil, errors.New("tidak ada shift aktif hari ini untuk poli yang dipilih")
		}
		return nil, nil, err
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
