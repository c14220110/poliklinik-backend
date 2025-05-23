package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/models"
	"github.com/c14220110/poliklinik-backend/ws"
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
//   - statusFilter: jika tidak kosong, filter berdasarkan Billing.id_status (1=Belum, 2=Selesai, 3=Dibatalkan)
// Jika salah satu kosong, ambil semua.
func (s *BillingService) GetBillingData(idPoliFilter, statusFilter string) ([]map[string]interface{}, error) {
	// Tentukan apakah statusFilter adalah "2" (Selesai)
	includeTanggalPembayaran := statusFilter == "2"

	// Query dasar
	query := `
			SELECT p.id_pasien, p.nama, rm.id_rm, pl.nama_poli, sb.status, rk.id_kunjungan
	`
	if includeTanggalPembayaran {
			query += ", b.updated_at AS tanggal_pembayaran"
	}
	query += `
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
			var tanggalPembayaran time.Time
			var scanArgs []interface{}

			// Tentukan argumen untuk Scan berdasarkan apakah tanggal_pembayaran disertakan
			scanArgs = append(scanArgs, &idPasien, &nama, &idRM, &namaPoli, &statusStr, &idKunjungan)
			if includeTanggalPembayaran {
					scanArgs = append(scanArgs, &tanggalPembayaran)
			}

			if err := rows.Scan(scanArgs...); err != nil {
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

			// Tambahkan tanggal_pembayaran jika statusFilter adalah "2"
			if includeTanggalPembayaran {
					record["tanggal_pembayaran"] = tanggalPembayaran
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
func (svc *BillingService) GetDetailBilling(idKunjungan int) (*models.DetailBilling, error) {
	// Validasi id_kunjungan
	var dummy int
	err := svc.DB.QueryRow("SELECT 1 FROM Riwayat_Kunjungan WHERE id_kunjungan = ?", idKunjungan).Scan(&dummy)
	if err == sql.ErrNoRows {
			return nil, ErrKunjunganNotFound
	}
	if err != nil {
			return nil, err
	}

	// Query untuk data utama dengan id_status dan updated_at
	query := `
			SELECT 
					p.nama AS nama_pasien,
					rk.id_rm,
					pol.nama_poli,
					COALESCE(k.nama, '') AS nama_dokter,
					COALESCE(td.harga, 0) AS biaya_dokter,
					COALESCE(ka.nama, '') AS karyawan_yang_ditugaskan,
					COALESCE(kb.nama, '') AS nama_administrasi,
					b.id_status,
					b.updated_at
			FROM Riwayat_Kunjungan rk
			JOIN Antrian a ON rk.id_antrian = a.id_antrian
			JOIN Poliklinik pol ON a.id_poli = pol.id_poli
			JOIN Pasien p ON a.id_pasien = p.id_pasien
			LEFT JOIN Assessment ass ON rk.id_assessment = ass.id_assessment
			LEFT JOIN Karyawan k ON ass.id_karyawan = k.id_karyawan
			LEFT JOIN Tarif_Dokter td ON k.id_karyawan = td.id_karyawan
			LEFT JOIN Billing b ON rk.id_kunjungan = b.id_kunjungan
			LEFT JOIN Billing_Assessment ba ON b.id_assessment = ba.id_assessment
			LEFT JOIN Karyawan ka ON ba.id_karyawan = ka.id_karyawan
			LEFT JOIN Karyawan kb ON b.id_karyawan = kb.id_karyawan
			WHERE rk.id_kunjungan = ?
	`
	row := svc.DB.QueryRow(query, idKunjungan)
	var detail models.DetailBilling
	var idStatus sql.NullInt64
	var updatedAt sql.NullTime
	err = row.Scan(
			&detail.NamaPasien,
			&detail.IDRM,
			&detail.NamaPoli,
			&detail.NamaDokter,
			&detail.BiayaDokter,
			&detail.KaryawanYangDitugaskan,
			&detail.NamaAdministrasi,
			&idStatus,
			&updatedAt,
	)
	if err == sql.ErrNoRows {
			return nil, ErrKunjunganNotFound
	}
	if err != nil {
			return nil, err
	}

	// Set WaktuDibayar jika id_status = 2
	if idStatus.Valid && idStatus.Int64 == 2 && updatedAt.Valid {
			waktuDibayarStr := updatedAt.Time.Format("2006-01-02 15:04:05")
			detail.WaktuDibayar = &waktuDibayarStr
	}

	// Query untuk daftar obat (unchanged)
	obatQuery := `
			SELECT 
					rs.section_type,
					COALESCE(o.nama, '') AS nama_obat,
					rs.jumlah,
					COALESCE(o.satuan, '') AS satuan,
					COALESCE(o.harga_satuan, 0) AS harga_satuan,
					rs.harga_total,
					rs.instruksi,
					COALESCE(rs.nama_racikan, '') AS nama_racikan,
					COALESCE(rs.jenis_kemasan, '') AS kemasan,
					rs.id_section
			FROM Resep_Section rs
			JOIN E_Resep er ON rs.id_resep = er.id_resep
			LEFT JOIN Komposisi k ON rs.id_section = k.id_section AND rs.section_type = 1
			LEFT JOIN Obat o ON k.id_obat = o.id_obat
			WHERE er.id_kunjungan = ?
	`
	rows, err := svc.DB.Query(obatQuery, idKunjungan)
	if err != nil {
			return nil, err
	}
	defer rows.Close()

	detail.Obat = []models.ObatDetail{}
	for rows.Next() {
			var obat models.ObatDetail
			var sectionType int
			var idSection int
			err := rows.Scan(
					&sectionType,
					&obat.NamaObat,
					&obat.Jumlah,
					&obat.Satuan,
					&obat.HargaSatuan,
					&obat.HargaTotal,
					&obat.Instruksi,
					&obat.NamaRacikan,
					&obat.Kemasan,
					&idSection,
			)
			if err != nil {
					return nil, err
			}
			if sectionType == 1 {
					obat.Keterangan = "obat resep"
			} else if sectionType == 2 {
					obat.Keterangan = "obat racikan"
					komposisiQuery := `
							SELECT 
									o.nama, 
									k.dosis, 
									o.satuan, 
									o.harga_satuan
							FROM Komposisi k
							JOIN Obat o ON k.id_obat = o.id_obat
							WHERE k.id_section = ?
					`
					komRows, err := svc.DB.Query(komposisiQuery, idSection)
					if err != nil {
							return nil, err
					}
					defer komRows.Close()
					obat.Komposisi = []models.KomposisiDetail{}
					for komRows.Next() {
							var kom models.KomposisiDetail
							err := komRows.Scan(&kom.NamaObat, &kom.Dosis, &kom.Satuan, &kom.HargaSatuan)
							if err != nil {
									return nil, err
							}
							obat.Komposisi = append(obat.Komposisi, kom)
					}
			}
			detail.Obat = append(detail.Obat, obat)
	}

	// Query untuk daftar tindakan (unchanged)
	tindakanQuery := `
			SELECT 
					ba.nama_tindakan, 
					ba.jumlah, 
					icd.harga AS harga_tindakan, 
					ba.total_harga_tindakan
			FROM Billing_Assessment ba
			JOIN ICD9_CM icd ON ba.id_icd9_cm = icd.id_icd9_cm
			WHERE ba.id_assessment = (SELECT id_assessment FROM Riwayat_Kunjungan WHERE id_kunjungan = ?)
	`
	rows, err = svc.DB.Query(tindakanQuery, idKunjungan)
	if err != nil {
			return nil, err
	}
	defer rows.Close()

	detail.Tindakan = []models.TindakanDetail{}
	for rows.Next() {
			var tindakan models.TindakanDetail
			err := rows.Scan(
					&tindakan.NamaTindakan,
					&tindakan.Jumlah,
					&tindakan.HargaTindakan,
					&tindakan.TotalHargaTindakan,
			)
			if err != nil {
					return nil, err
			}
			detail.Tindakan = append(detail.Tindakan, tindakan)
	}

	return &detail, nil
}

var (
	ErrKunjunganNotFound = errors.New("kunjungan not found")
)


func (s *BillingService) BayarTagihan(
	idKunjungan int,
	tipePembayaran string,
) (map[string]interface{}, error) {

	tx, err := s.DB.Begin()
	if err != nil {
			return nil, fmt.Errorf("gagal memulai transaksi: %v", err)
	}
	defer tx.Rollback()

	// 1) Ambil record Billing berdasarkan id_kunjungan
	var (
			idBilling    int
			idAntrian    sql.NullInt64
			idAssessment sql.NullInt64
			currentStatus int
	)
	err = tx.QueryRow(`
			SELECT id_billing, id_antrian, id_assessment, id_status
			FROM Billing
			WHERE id_kunjungan = ?`,
			idKunjungan,
	).Scan(&idBilling, &idAntrian, &idAssessment, &currentStatus)
	if err != nil {
			if err == sql.ErrNoRows {
					return nil, fmt.Errorf("tagihan tidak ditemukan")
			}
			return nil, fmt.Errorf("gagal mengambil data billing: %v", err)
	}
	if currentStatus == 2 {
			return nil, fmt.Errorf("tagihan sudah dibayar")
	}

	// 2) Hitung harga dokter (jika ada assessment)
	var hargaDokter float64
	if idAssessment.Valid {
			var idKaryawanAssess int
			if err = tx.QueryRow(
					`SELECT id_karyawan FROM Assessment WHERE id_assessment = ?`,
					idAssessment.Int64,
			).Scan(&idKaryawanAssess); err != nil && err != sql.ErrNoRows {
					return nil, fmt.Errorf("gagal mengambil id_karyawan: %v", err)
			}
			if err == nil {
					tx.QueryRow(
							`SELECT harga FROM Tarif_Dokter WHERE id_karyawan = ?`,
							idKaryawanAssess,
					).Scan(&hargaDokter) // nol jika tidak ada
			}
	}

	// 3) Hitung total obat
	var totalObat float64
	if idKunjungan > 0 {
			tx.QueryRow(
					`SELECT COALESCE(SUM(total_harga),0) FROM E_Resep WHERE id_kunjungan = ?`,
					idKunjungan,
			).Scan(&totalObat)
	}

	// 4) Hitung total tindakan
	var totalTindakan float64
	if idAssessment.Valid {
			tx.QueryRow(
					`SELECT COALESCE(SUM(total_harga_tindakan),0) 
					 FROM Billing_Assessment 
					 WHERE id_assessment = ?`,
					idAssessment.Int64,
			).Scan(&totalTindakan)
	}

	// 5) Total keseluruhan
	total := hargaDokter + totalObat + totalTindakan

	// 6) Update Billing (by PK id_billing)
	if _, err = tx.Exec(`
			UPDATE Billing
			SET tipe_pembayaran = ?, total = ?, id_status = 2
			WHERE id_billing = ?`,
			tipePembayaran, total, idBilling,
	); err != nil {
			return nil, fmt.Errorf("gagal memperbarui billing: %v", err)
	}

	// 7) Ambil data utk broadcast WebSocket
	var (
			wsIdPasien  int64
			wsNamaPasien, wsNomorRM, wsNamaPoli string
			wsUpdatedAt time.Time
	)
	err = tx.QueryRow(`
			SELECT A.id_pasien, P.nama, RK.id_rm, Pol.nama_poli, B.updated_at
			FROM Billing B
			JOIN Antrian A ON B.id_antrian = A.id_antrian
			JOIN Pasien P   ON A.id_pasien  = P.id_pasien
			JOIN Riwayat_Kunjungan RK ON B.id_kunjungan = RK.id_kunjungan
			JOIN Kunjungan_Poli KP   ON RK.id_kunjungan = KP.id_kunjungan
			JOIN Poliklinik Pol      ON KP.id_poli = Pol.id_poli
			WHERE B.id_billing = ?`,
			idBilling,
	).Scan(&wsIdPasien, &wsNamaPasien, &wsNomorRM, &wsNamaPoli, &wsUpdatedAt)
	if err != nil {
			return nil, fmt.Errorf("gagal mengambil data untuk WebSocket: %v", err)
	}

	// commit
	if err = tx.Commit(); err != nil {
			return nil, fmt.Errorf("gagal commit transaksi: %v", err)
	}

	// broadcast pertama
	payload := map[string]interface{}{
			"type": "billing_update",
			"data": map[string]interface{}{
					"id_kunjungan": idKunjungan,
					"id_pasien":    wsIdPasien,
					"nama_pasien":  wsNamaPasien,
					"id_rm":        wsNomorRM,
					"nama_poli":    wsNamaPoli,
					"status":       "Selesai",
			},
	}
	msg, _ := json.Marshal(payload)
	ws.HubInstance.Broadcast <- msg

	// broadcast kedua
	payload2 := map[string]interface{}{
			"type": "riwayat_update",
			"data": map[string]interface{}{
					"nama_pasien":        wsNamaPasien,
					"tanggal_pembayaran": wsUpdatedAt,
					"nama_poli":          wsNamaPoli,
					"id_kunjungan":       idKunjungan,
			},
	}
	msg2, _ := json.Marshal(payload2)
	ws.HubInstance.Broadcast <- msg2

	// 8) prepare response
	result := map[string]interface{}{
			"tarif_dokter":   hargaDokter,
			"total_obat":     totalObat,
			"total_tindakan": totalTindakan,
			"total":          total,
	}
	return result, nil
}
