package services

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
)

type BillingService struct {
	DB *sql.DB
}

func NewBillingService(db *sql.DB) *BillingService {
	return &BillingService{DB: db}
}

// GetBillingData mengambil data billing dengan join ke Pasien, Rekam_Medis, dan Poliklinik.
// Filter:
//   - idPoliFilter: jika tidak kosong, filter berdasarkan poliklinik
//   - statusFilter: jika tidak kosong, filter berdasarkan Billing.id_status (1=Belum, 2=Diproses, 3=Selesai, 4=Dibatalkan)
// Jika salah satu kosong, ambil semua.
func (s *BillingService) GetBillingData(idPoliFilter, statusFilter string) ([]map[string]interface{}, error) {
	// Query dengan join ke tabel terkait, termasuk Status_Billing dan Riwayat_Kunjungan untuk mengambil id_kunjungan.
	query := `
		SELECT p.id_pasien, p.nama, rm.id_rm, pl.nama_poli, sb.status, rk.id_kunjungan
		FROM Billing b
		JOIN Status_Billing sb ON b.id_status = sb.id_status
		JOIN Riwayat_Kunjungan rk ON b.id_kunjungan = rk.id_kunjungan
		JOIN Rekam_Medis rm ON rk.id_rm = rm.id_rm
		JOIN Pasien p ON rm.id_pasien = p.id_pasien
		JOIN Kunjungan_Poli kp ON rk.id_kunjungan = kp.id_kunjungan
		JOIN Poliklinik pl ON kp.id_poli = pl.id_poli
	`
	conditions := []string{}
	args := []interface{}{}

	// Filter data Billing hanya untuk hari ini (berdasarkan created_at Billing)
	today := time.Now().Format("2006-01-02")
	conditions = append(conditions, "DATE(b.created_at) = ?")
	args = append(args, today)

	// Filter berdasarkan id_poli jika disediakan
	if idPoliFilter != "" {
		idPoli, err := strconv.Atoi(idPoliFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid id_poli value: %v", err)
		}
		conditions = append(conditions, "pl.id_poli = ?")
		args = append(args, idPoli)
	}

	// Filter berdasarkan status jika disediakan
	if statusFilter != "" {
		st, err := strconv.Atoi(statusFilter)
		if err != nil {
			return nil, fmt.Errorf("invalid status value: %v", err)
		}
		conditions = append(conditions, "b.id_status = ?")
		args = append(args, st)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY p.id_pasien DESC"

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var idPasien int
		var nama string
		var idRM string
		var namaPoli string
		var statusStr string
		var idKunjungan int
		if err := rows.Scan(&idPasien, &nama, &idRM, &namaPoli, &statusStr, &idKunjungan); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}
		record := map[string]interface{}{
			"id_pasien":    idPasien,
			"nama_pasien":  nama,
			"id_rm":        idRM,
			"nama_poli":    namaPoli,
			"status":       statusStr,
			"id_kunjungan": idKunjungan,
		}
		results = append(results, record)
	}
	return results, nil
}

func (svc *BillingService) SaveBillingAssessment(
	idAntrian int,
	idAssessment int,
	idKaryawanJWT int,
	in models.InputBillingRequest,
) error {
	tx, err := svc.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 1. Memeriksa kecocokan assessment dan antrian
	var idPasienFromAss, idPasienFromAntrian int
	if err = tx.QueryRow(
		`SELECT id_pasien FROM Assessment WHERE id_assessment = ?`,
		idAssessment).Scan(&idPasienFromAss); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("assessment not found")
		}
		return err
	}
	if err = tx.QueryRow(
		`SELECT id_pasien FROM Antrian WHERE id_antrian = ?`,
		idAntrian).Scan(&idPasienFromAntrian); err != nil {
		return err
	}
	if idPasienFromAss != idPasienFromAntrian {
		return fmt.Errorf("assessment does not belong to given antrian")
	}

	// 2. Menyiapkan prepared statements
	stmtSel, err := tx.Prepare(`SELECT display, harga FROM ICD9_CM WHERE id_icd9_cm = ?`)
	if err != nil {
		return err
	}
	defer stmtSel.Close()

	stmtIns, err := tx.Prepare(`
		INSERT INTO Billing_Assessment
		  (id_assessment, id_karyawan, id_icd9_cm, nama_tindakan,
		   jumlah, total_harga_tindakan, created_at)
		VALUES (?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmtIns.Close()

	// Menentukan id_karyawan (PIC) — jika body diisi 0, gunakan id dari JWT
	picID := in.NamaPICTindakan
	if picID == 0 {
		picID = idKaryawanJWT
	}

	// 3. Memproses setiap tindakan
	for _, td := range in.Tindakan {
		var display string
		var harga float64
		if err = stmtSel.QueryRow(td.Tindakan).Scan(&display, &harga); err != nil {
			return fmt.Errorf("icd9_cm %s not found", td.Tindakan)
		}

		// Menghitung total_harga_tindakan
		totalHarga := float64(td.Jumlah) * harga

		if _, err = stmtIns.Exec(
			idAssessment,
			picID,
			td.Tindakan,
			display,
			td.Jumlah,
			totalHarga,
			time.Now(),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}