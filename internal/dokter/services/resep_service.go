package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/dokter/models"
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

func (s *ResepService) GetRiwayatKunjunganByPasien(idPasien int) ([]models.RiwayatKunjungan, error) {
    query := `
        SELECT 
            rk.id_kunjungan,
            rk.created_at AS tanggal,
            p.nama_poli AS tujuan_poli,
            a.nomor_antrian,
            a.keluhan_utama,
            icd10.display AS hasil_diagnosa,
            icd9.display AS tindakan,
            rk.id_resep,
            rk.id_assessment
        FROM Riwayat_Kunjungan rk
        JOIN Antrian a ON rk.id_antrian = a.id_antrian
        JOIN Kunjungan_Poli kp ON rk.id_kunjungan = kp.id_kunjungan
        JOIN Poliklinik p ON kp.id_poli = p.id_poli
        LEFT JOIN Assessment ass ON rk.id_assessment = ass.id_assessment
        LEFT JOIN ICD10 icd10 ON ass.id_icd10 = icd10.id_icd10
        LEFT JOIN Billing_Assessment ba ON ass.id_assessment = ba.id_assessment
        LEFT JOIN ICD9_CM icd9 ON ba.id_icd9_cm = icd9.id_icd9_cm
        WHERE a.id_pasien = ?
        ORDER BY rk.id_kunjungan
    `
    rows, err := s.DB.Query(query, idPasien)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []models.RiwayatKunjungan
    var currentVisit *models.RiwayatKunjungan
    var lastIDKunjungan int

    for rows.Next() {
        var idKunjungan int
        var tanggal time.Time
        var tujuanPoli string
        var nomorAntrian int
        var keluhanUtama string
        var hasilDiagnosa sql.NullString
        var tindakan sql.NullString
        var idResep sql.NullInt64
        var idAssessment sql.NullInt64

        err := rows.Scan(
            &idKunjungan,
            &tanggal,
            &tujuanPoli,
            &nomorAntrian,
            &keluhanUtama,
            &hasilDiagnosa,
            &tindakan,
            &idResep,
            &idAssessment,
        )
        if err != nil {
            return nil, err
        }

        // Start a new visit if id_kunjungan changes
        if currentVisit == nil || idKunjungan != lastIDKunjungan {
            if currentVisit != nil {
                result = append(result, *currentVisit)
            }
            currentVisit = &models.RiwayatKunjungan{
                IDKunjungan:   idKunjungan,
                Tanggal:       tanggal.Format("02/01/2006"), // DD/MM/YYYY
                TujuanPoli:    tujuanPoli,
                NomorAntrian:  nomorAntrian,
                KeluhanUtama:  keluhanUtama,
                HasilDiagnosa: hasilDiagnosa.String, // Empty string if null
                Tindakan:      []string{},
            }
            if idResep.Valid {
                idResepInt := int(idResep.Int64)
                currentVisit.IDResep = &idResepInt
            }
            if idAssessment.Valid {
                idAssessmentInt := int(idAssessment.Int64)
                currentVisit.IDAssessment = &idAssessmentInt
            }
            lastIDKunjungan = idKunjungan
        }

        // Append tindakan if present
        if tindakan.Valid {
            currentVisit.Tindakan = append(currentVisit.Tindakan, tindakan.String)
        }
    }

    // Append the last visit
    if currentVisit != nil {
        result = append(result, *currentVisit)
    }

    return result, nil
}