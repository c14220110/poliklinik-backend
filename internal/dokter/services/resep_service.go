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

// CreateResep:
//   • Memastikan id_kunjungan valid
//   • Menghitung total harga dari setiap section (grandTotal)
//   • Menulis E_Resep, Resep_Section, Komposisi   —  semuanya di dalam transaksi
//   • Menautkan id_resep ke Riwayat_Kunjungan
func (s *ResepService) CreateResep(req dmodels.ResepRequest) (int64, error) {
    tx, err := s.DB.Begin()
    if err != nil {
        return 0, err
    }

    // 0. Validasi id_kunjungan
    var dummy int
    if err := tx.QueryRow("SELECT 1 FROM Riwayat_Kunjungan WHERE id_kunjungan = ?", req.IDKunjungan).Scan(&dummy); err != nil {
        tx.Rollback()
        if err == sql.ErrNoRows {
            return 0, ErrKunjunganNotFound
        }
        return 0, err
    }

    // 1. Hitung grand total
    var grandTotal float64
    for _, sec := range req.Sections {
        grandTotal += sec.HargaTotal
    }

    // 2. Insert ke E_Resep
    resResep, err := tx.Exec(
        `INSERT INTO E_Resep (id_kunjungan, id_karyawan, created_at, total_harga)
         VALUES (?,?,?,?)`,
        req.IDKunjungan, req.IDKaryawan, time.Now(), grandTotal,
    )
    if err != nil {
        tx.Rollback(); return 0, err
    }
    idResep, err := resResep.LastInsertId()
    if err != nil { tx.Rollback(); return 0, err }

    // 3. Loop section
    for _, sec := range req.Sections {
        // section_type mapping
        var secType int
        switch sec.SectionType {
        case "obat":
            secType = 1
        case "racikan":
            secType = 2
        default:
            tx.Rollback(); return 0, errors.New("invalid section_type")
        }

        resSec, err := tx.Exec(
            `
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

        // 4. Komposisi
        if secType == 1 {
            // obat tunggal → dosis = jumlah
            if sec.IDObat == nil {
                tx.Rollback(); return 0, errors.New("id_obat required for section_type 'obat'")
            }
            if _, err := tx.Exec(`INSERT INTO Komposisi (id_section, id_obat, dosis) VALUES (?,?,?)`, sectionID, *sec.IDObat, sec.Jumlah); err != nil {
                tx.Rollback(); return 0, err
            }
        } else {
            for _, cmp := range sec.Komposisi {
                if _, err := tx.Exec(`INSERT INTO Komposisi (id_section, id_obat, dosis) VALUES (?,?,?)`, sectionID, cmp.IDObat, cmp.Dosis); err != nil {
                    tx.Rollback(); return 0, err
                }
            }
        }
    }

    // 5. Update Riwayat_Kunjungan dengan id_resep
    if _, err := tx.Exec(`UPDATE Riwayat_Kunjungan SET id_resep = ? WHERE id_kunjungan = ?`, idResep, req.IDKunjungan); err != nil {
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

func (s *ResepService) GetRiwayatKunjunganByPasien(idPasien int) ([]map[string]interface{}, error) {
    query := `
        SELECT 
            rk.id_kunjungan,
            rk.created_at AS tanggal,
            p.nama_poli AS tujuan_poli,
            a.nomor_antrian,
            a.keluhan_utama,
            icd10.display AS hasil_diagnosa,
            rk.id_resep,
            rk.id_assessment
        FROM Riwayat_Kunjungan rk
        JOIN Rekam_Medis rm ON rk.id_rm = rm.id_rm
        JOIN Pasien pas ON rm.id_pasien = pas.id_pasien
        LEFT JOIN Kunjungan_Poli kp ON rk.id_kunjungan = kp.id_kunjungan
        LEFT JOIN Poliklinik p ON kp.id_poli = p.id_poli
        LEFT JOIN Antrian a ON rk.id_antrian = a.id_antrian
        LEFT JOIN Assessment ass ON rk.id_assessment = ass.id_assessment
        LEFT JOIN ICD10 icd10 ON ass.id_icd10 = icd10.id_icd10
        WHERE pas.id_pasien = ?
        ORDER BY rk.created_at DESC
    `
    rows, err := s.DB.Query(query, idPasien)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var riwayatKunjungan []map[string]interface{}
    for rows.Next() {
        var kunjungan struct {
            ID_Kunjungan   int
            Tanggal        string
            Tujuan_Poli    string
            Nomor_Antrian  int
            Keluhan_Utama  string
            Hasil_Diagnosa string
            ID_Resep       int
            ID_Assessment  int
        }
        err := rows.Scan(
            &kunjungan.ID_Kunjungan,
            &kunjungan.Tanggal,
            &kunjungan.Tujuan_Poli,
            &kunjungan.Nomor_Antrian,
            &kunjungan.Keluhan_Utama,
            &kunjungan.Hasil_Diagnosa,
            &kunjungan.ID_Resep,
            &kunjungan.ID_Assessment,
        )
        if err != nil {
            return nil, err
        }

        // Ambil tindakan untuk assessment ini
        tindakanQuery := `
            SELECT icd9.display
            FROM Billing_Assessment ba
            JOIN ICD9_CM icd9 ON ba.id_icd9_cm = icd9.id_icd9_cm
            WHERE ba.id_assessment = ?
        `
        tindakanRows, err := s.DB.Query(tindakanQuery, kunjungan.ID_Assessment)
        if err != nil {
            return nil, err
        }
        defer tindakanRows.Close()

        var tindakan []string
        for tindakanRows.Next() {
            var display string
            if err := tindakanRows.Scan(&display); err != nil {
                return nil, err
            }
            tindakan = append(tindakan, display)
        }

        record := map[string]interface{}{
            "id_kunjungan":    kunjungan.ID_Kunjungan,
            "tanggal":         kunjungan.Tanggal,
            "tujuan_poli":     kunjungan.Tujuan_Poli,
            "nomor_antrian":   kunjungan.Nomor_Antrian,
            "keluhan_utama":   kunjungan.Keluhan_Utama,
            "hasil_diagnosa":  kunjungan.Hasil_Diagnosa,
            "tindakan":        tindakan,
            "id_resep":        kunjungan.ID_Resep,
            "id_assessment":   kunjungan.ID_Assessment,
        }
        riwayatKunjungan = append(riwayatKunjungan, record)
    }
    return riwayatKunjungan, nil
}