package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dmodels "github.com/c14220110/poliklinik-backend/internal/dokter/models"
)

var (
	ErrKunjunganNotFound = errors.New("kunjungan tidak ditemukan")
)

type ResepService struct{ DB *sql.DB }

func NewResepService(db *sql.DB) *ResepService { return &ResepService{DB: db} }

// CreateResep menyimpan resep obat & racikan, kemudian menautkannya
// ke tabel Riwayat_Kunjungan.id_resep  (transaksi penuh)
func (s *ResepService) CreateResep(req dmodels.ResepRequest) (int64, error) {

	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}

	// Pastikan id_kunjungan valid
	var exists int
	if err := tx.QueryRow("SELECT 1 FROM Riwayat_Kunjungan WHERE id_kunjungan = ?", req.IDKunjungan).Scan(&exists); err != nil {
		tx.Rollback()
		if err == sql.ErrNoRows {
			return 0, ErrKunjunganNotFound
		}
		return 0, err
	}

	// 1. Insert ke E_Resep
	resResep, err := tx.Exec(`
		INSERT INTO E_Resep (id_kunjungan, id_karyawan, created_at, total_harga)
		VALUES (?,?,?,?)`,
		req.IDKunjungan, req.IDKaryawan, time.Now(), req.TotalHarga)
	if err != nil {
		tx.Rollback(); return 0, err
	}
	idResep, err := resResep.LastInsertId()
	if err != nil { tx.Rollback(); return 0, err }

	// 2. Insert setiap section
	for _, sec := range req.Sections {

		// konversi section_type => tinyint (1=obat,2=racikan)
		var secType int
		switch sec.SectionType {
		case "obat":
			secType = 1
		case "racikan":
			secType = 2
		default:
			tx.Rollback()
			return 0, errors.New("invalid section_type")
		}

		// Resep_Section
		resSec, err := tx.Exec(`
    INSERT INTO Resep_Section
      (id_resep, section_type, nama_racikan, jumlah, jenis_kemasan, instruksi, harga_total)
    VALUES (?,?,?,?,?,?,?)`,
    idResep,
    secType,
    sql.NullString{String: sec.NamaRacikan, Valid: secType == 2},
    sec.Jumlah,
    sql.NullString{String: sec.Kemasan, Valid: secType == 2},
    sec.Instruksi,
    sec.HargaTotal,
)
		if err != nil { tx.Rollback(); return 0, err }

		sectionID, err := resSec.LastInsertId()
		if err != nil { tx.Rollback(); return 0, err }

		// 3. Komposisi
		if secType == 1 {
			// obat tunggal → satu baris komposisi dengan dosis = jumlah
			_, err = tx.Exec(`
				INSERT INTO Komposisi (id_section, id_obat, dosis)
				VALUES (?,?,?)`,
				sectionID, *sec.IDObat, sec.Jumlah)
			if err != nil { tx.Rollback(); return 0, err }
		} else { // racikan
			for _, cmp := range sec.Komposisi {
				_, err = tx.Exec(`
					INSERT INTO Komposisi (id_section, id_obat, dosis)
					VALUES (?,?,?)`,
					sectionID, cmp.IDObat, cmp.Dosis)
				if err != nil { tx.Rollback(); return 0, err }
			}
		}
	}

	// 4. Update Riwayat_Kunjungan.id_resep
	if _, err := tx.Exec(
		`UPDATE Riwayat_Kunjungan SET id_resep = ? WHERE id_kunjungan = ?`,
		idResep, req.IDKunjungan); err != nil {
		tx.Rollback(); return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return idResep, nil
}


// GetObatList menampilkan daftar obat dengan pencarian nama + pagination.
// • q     : string pencarian, case‑insensitive, boleh kosong
// • limit : jumlah baris per halaman (default 20, max 100)
// • page  : halaman dimulai dari 1 (default 1)
func (s *ResepService) GetObatList(q string, limit, page int) ([]map[string]interface{}, error) {

	if limit <= 0 { limit = 20 }
	if limit > 100 { limit = 100 }
	if page  <= 0 { page  = 1  }
	offset := (page - 1) * limit

	baseQuery := `
		SELECT id_obat, nama, harga_satuan, satuan, jenis, stock
		FROM Obat
	`
	conds  := []string{}
	params := []interface{}{}

	if q != "" {
		conds  = append(conds, "LOWER(nama) LIKE ?")
		params = append(params, "%"+strings.ToLower(q)+"%")
	}

	query := baseQuery
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY id_obat"
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	rows, err := s.DB.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var (
			id int
			nama, satuan, jenis string
			harga float64
			stock int
		)
		if err := rows.Scan(&id, &nama, &harga, &satuan, &jenis, &stock); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}
		list = append(list, map[string]interface{}{
			"id_obat":      id,
			"nama":         nama,
			"harga_satuan": harga,
			"satuan":       satuan,
			"jenis":        jenis,
			"stock":        stock,
		})
	}
	return list, nil
}