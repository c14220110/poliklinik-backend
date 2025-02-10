package services

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Dokter struct {
	ID_Dokter    int
	Nama         string
	Username     string
	Password     string
	Spesialisasi string
	CreatedAt    time.Time
}

type ShiftKaryawan struct {
	ID_Shift_Karyawan int
	ID_Poli           int
	ID_Shift          int
	ID_Karyawan       int
	Tanggal           time.Time
	Jam_Mulai         time.Time
	Jam_Selesai       time.Time
}

type DokterService struct {
	DB *sql.DB
}

func NewDokterService(db *sql.DB) *DokterService {
	return &DokterService{DB: db}
}

func (ds *DokterService) AuthenticateDokterUsingKaryawan(username, password string, selectedPoli int) (*Dokter, *ShiftKaryawan, error) {
	var dokter Dokter
	query := "SELECT ID_Karyawan, Nama, Username, Password, '' as Spesialisasi, Created_At FROM Karyawan WHERE Username = ?"
	err := ds.DB.QueryRow(query, username).Scan(&dokter.ID_Dokter, &dokter.Nama, &dokter.Username, &dokter.Password, &dokter.Spesialisasi, &dokter.CreatedAt)
	if err != nil {
		return nil, nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(dokter.Password), []byte(password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	var roleName string
	roleQuery := `
		SELECT r.Nama_Role 
		FROM Detail_Role_Karyawan drk 
		JOIN Role r ON drk.ID_Role = r.ID_Role 
		WHERE drk.ID_Karyawan = ?
		LIMIT 1
	`
	err = ds.DB.QueryRow(roleQuery, dokter.ID_Dokter).Scan(&roleName)
	if err != nil {
		return nil, nil, errors.New("failed to retrieve role")
	}
	if roleName != "Dokter" {
		return nil, nil, errors.New("user is not a Dokter")
	}

	//now := time.Now()
	var shift ShiftKaryawan
	shiftQuery := `
		SELECT sk.ID_Shift_Karyawan, sk.ID_Poli, sk.ID_Shift, sk.ID_Karyawan, sk.Tanggal,
		       s.Jam_Mulai, s.Jam_Selesai
		FROM Shift_Karyawan sk
		JOIN Shift s ON sk.ID_Shift = s.ID_Shift
		WHERE sk.ID_Karyawan = ? AND sk.ID_Poli = ? AND sk.Tanggal = CURDATE()
		  AND (
		    (s.Jam_Mulai < s.Jam_Selesai AND CURTIME() BETWEEN s.Jam_Mulai AND s.Jam_Selesai)
		    OR (s.Jam_Mulai > s.Jam_Selesai AND (CURTIME() >= s.Jam_Mulai OR CURTIME() < s.Jam_Selesai))
		  )
		LIMIT 1
	`
	err = ds.DB.QueryRow(shiftQuery, dokter.ID_Dokter, selectedPoli).Scan(&shift.ID_Shift_Karyawan, &shift.ID_Poli, &shift.ID_Shift, &shift.ID_Karyawan, &shift.Tanggal, &shift.Jam_Mulai, &shift.Jam_Selesai)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.New("no active shift for this Dokter on the selected poliklinik")
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
		var idRM sql.NullInt64
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
			record["id_rm"] = idRM.Int64
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
