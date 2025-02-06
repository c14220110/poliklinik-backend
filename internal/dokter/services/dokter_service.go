package services

import (
	"database/sql"
	"errors"

	//"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/c14220110/poliklinik-backend/internal/dokter/models"
)

// DokterService menangani operasi terkait dokter, termasuk autentikasi dan (baru) pengambilan antrian.
type DokterService struct {
	DB *sql.DB
}

func NewDokterService(db *sql.DB) *DokterService {
	return &DokterService{DB: db}
}

// CreateDokter mendaftarkan dokter baru dengan meng-hash password-nya.
func (s *DokterService) CreateDokter(d models.Dokter) (int64, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(d.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	query := `INSERT INTO Dokter (Nama, Username, Password, Spesialisasi) VALUES (?, ?, ?, ?)`
	result, err := s.DB.Exec(query, d.Nama, d.Username, hashedPassword, d.Spesialisasi)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// AuthenticateDokter memvalidasi login dokter dengan cek shift aktif (lihat revisi sebelumnya).
// (Fungsi ini sudah ada dan tidak berubah, jadi tetap seperti sebelumnya.)
func (s *DokterService) AuthenticateDokter(username, password string, selectedPoli int) (*models.Dokter, *models.ShiftDokter, error) {
	var dokter models.Dokter
	queryDokter := `SELECT ID_Dokter, Nama, Username, Password, Spesialisasi FROM Dokter WHERE Username = ?`
	err := s.DB.QueryRow(queryDokter, username).Scan(&dokter.ID_Dokter, &dokter.Nama, &dokter.Username, &dokter.Password, &dokter.Spesialisasi)
	if err != nil {
		return nil, nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(dokter.Password), []byte(password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	// Cek shift aktif di Shift_Dokter
	now := time.Now()
	var shift models.ShiftDokter
	queryShift := `
		SELECT ID_Shift, ID_Dokter, ID_Poli, ID_Management, Jam_Mulai, Jam_Selesai 
		FROM Shift_Dokter 
		WHERE ID_Dokter = ? AND Jam_Mulai <= ? AND Jam_Selesai >= ?
	`
	err = s.DB.QueryRow(queryShift, dokter.ID_Dokter, now, now).Scan(&shift.ID_Shift, &shift.ID_Dokter, &shift.ID_Poli, &shift.ID_Management, &shift.Jam_Mulai, &shift.Jam_Selesai)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, errors.New("dokter tidak memiliki shift aktif")
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
