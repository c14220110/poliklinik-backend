package services

import (
	"database/sql"
	"log"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
)

type DashboardService struct {
	DB *sql.DB
}

func NewDashboardService(db *sql.DB) *DashboardService {
	return &DashboardService{DB: db}
}

// GetDashboardData retrieves all metrics for the management dashboard.
// idPoli: pointer to poli ID filter (nil for all)
// start, end: date range (inclusive)
func (svc *DashboardService) GetDashboardData(idPoli *int, start, end time.Time) (models.DashboardData, error) {
	var d models.DashboardData
	// extend end to end of day
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	// helper to count by status or total
	countQuery := func(status *int) (int, error) {
		var (
			q      string
			params []interface{}
		)
		if status != nil {
			// status-specific count using updated_at
			q = "SELECT COUNT(*) FROM Antrian WHERE id_status = ? AND updated_at BETWEEN ? AND ?"
			params = append(params, *status, start, end)
			if idPoli != nil {
				q += " AND id_poli = ?"
				params = append(params, *idPoli)
			}
		} else {
			// total count using created_at
			q = "SELECT COUNT(*) FROM Antrian WHERE created_at BETWEEN ? AND ?"
			params = append(params, start, end)
			if idPoli != nil {
				q += " AND id_poli = ?"
				params = append(params, *idPoli)
			}
		}
		log.Printf("DEBUG countQuery: q=%s params=%v", q, params)
		var cnt int
		if err := svc.DB.QueryRow(q, params...).Scan(&cnt); err != nil {
			return 0, err
		}
		return cnt, nil
	}

	// 1. Pasien Dibatalkan (status = 7)
	cnt, err := countQuery(ptrInt(7))
	if err != nil {
		return d, err
	}
	d.PasienDibatalkan = cnt

	// 2. Pasien Konsultasi (status = 5)
	cnt, err = countQuery(ptrInt(5))
	if err != nil {
		return d, err
	}
	d.PasienKonsultasi = cnt

	// 3. Pasien Menunggu (status = 1)
	cnt, err = countQuery(ptrInt(1))
	if err != nil {
		return d, err
	}
	d.PasienMenunggu = cnt

	// 4. Total Pasien (all)
	cnt, err = countQuery(nil)
	if err != nil {
		return d, err
	}
	d.TotalPasien = cnt

	// 5. Jumlah Tenaga Kesehatan
	if err := svc.DB.QueryRow("SELECT COUNT(*) FROM Karyawan WHERE deleted_at IS NULL").Scan(&d.KaryawanAktif); err != nil {
		return d, err
	}
	if err := svc.DB.QueryRow("SELECT COUNT(*) FROM Karyawan WHERE deleted_at IS NOT NULL").Scan(&d.KaryawanNonAktif); err != nil {
		return d, err
	}

	// 6. Tren Penyakit
	trendQ := "SELECT i.display, COUNT(*) FROM Assessment a JOIN ICD10 i ON a.id_icd10 = i.id_icd10 WHERE a.created_at BETWEEN ? AND ?"
	trendArgs := []interface{}{start, end}
	if idPoli != nil {
		trendQ += " AND a.id_poli = ?"
		trendArgs = append(trendArgs, *idPoli)
	}
	trendQ += " GROUP BY i.display"
	log.Printf("DEBUG trenPenyakit: q=%s args=%v", trendQ, trendArgs)
	rows, err := svc.DB.Query(trendQ, trendArgs...)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var t models.PenyakitTrend
		if err := rows.Scan(&t.Display, &t.Count); err != nil {
			return d, err
		}
		d.TrenPenyakit = append(d.TrenPenyakit, t)
	}

	// 7. Pendapatan Total
	billQ := "SELECT COALESCE(SUM(b.total),0) FROM Billing b JOIN Antrian a ON b.id_antrian=a.id_antrian WHERE b.created_at BETWEEN ? AND ?"
	billArgs := []interface{}{start, end}
	if idPoli != nil {
		billQ += " AND a.id_poli = ?"
		billArgs = append(billArgs, *idPoli)
	}
	log.Printf("DEBUG pendapatanTotal: q=%s args=%v", billQ, billArgs)
	if err := svc.DB.QueryRow(billQ, billArgs...).Scan(&d.PendapatanTotal); err != nil {
		return d, err
	}

	// 8. Rata-rata Pendapatan
	days := end.Sub(start).Hours()/24 + 1
	if days > 0 {
		d.PendapatanRataRata = d.PendapatanTotal / days
	}

	// 9. Kunjungan Terbanyak
	kunjQ := "SELECT a.id_poli, COUNT(*) FROM Antrian a WHERE a.created_at BETWEEN ? AND ?"
	kunjArgs := []interface{}{start, end}
	if idPoli != nil {
		kunjQ += " AND a.id_poli = ?"
		kunjArgs = append(kunjArgs, *idPoli)
	}
	kunjQ += " GROUP BY a.id_poli ORDER BY COUNT(*) DESC"
	log.Printf("DEBUG kunjunganTerbanyak: q=%s args=%v", kunjQ, kunjArgs)
	krows, err := svc.DB.Query(kunjQ, kunjArgs...)
	if err != nil {
		return d, err
	}
	defer krows.Close()
	for krows.Next() {
		var pc models.PoliCount
		if err := krows.Scan(&pc.IDPoli, &pc.Count); err != nil {
			return d, err
		}
		d.KunjunganTerbanyak = append(d.KunjunganTerbanyak, pc)
	}

	// 10. Kunjungan Harian
	hQ := "SELECT DAYNAME(created_at) AS period, COUNT(*) FROM Antrian WHERE created_at BETWEEN ? AND ?"
	hArgs := []interface{}{start, end}
	if idPoli != nil {
		hQ += " AND id_poli = ?"
		hArgs = append(hArgs, *idPoli)
	}
	hQ += " GROUP BY DAYNAME(created_at)"
	hRows, err := svc.DB.Query(hQ, hArgs...)
	if err != nil {
		return d, err
	}
	defer hRows.Close()
	for hRows.Next() {
		var tc models.TimeCount
		if err := hRows.Scan(&tc.Period, &tc.Count); err != nil {
			return d, err
		}
		d.KunjunganHarian = append(d.KunjunganHarian, tc)
	}

	// 11. Kunjungan Mingguan
	wQ := "SELECT YEARWEEK(created_at,1) AS period, COUNT(*) FROM Antrian WHERE created_at BETWEEN ? AND ?"
	wArgs := []interface{}{start, end}
	if idPoli != nil {
		wQ += " AND id_poli = ?"
		wArgs = append(wArgs, *idPoli)
	}
	wQ += " GROUP BY YEARWEEK(created_at,1)"
	wRows, err := svc.DB.Query(wQ, wArgs...)
	if err != nil {
		return d, err
	}
	defer wRows.Close()
	for wRows.Next() {
		var tc models.TimeCount
		if err := wRows.Scan(&tc.Period, &tc.Count); err != nil {
			return d, err
		}
		d.KunjunganMingguan = append(d.KunjunganMingguan, tc)
	}

	// 12. Kunjungan Bulanan
	mQ := "SELECT DATE_FORMAT(created_at,'%Y-%m') AS period, COUNT(*) FROM Antrian WHERE created_at BETWEEN ? AND ?"
	mArgs := []interface{}{start, end}
	if idPoli != nil {
		mQ += " AND id_poli = ?"
		mArgs = append(mArgs, *idPoli)
	}
	mQ += " GROUP BY DATE_FORMAT(created_at,'%Y-%m') ORDER BY period"
	mRows, err := svc.DB.Query(mQ, mArgs...)
	if err != nil {
		return d, err
	}
	defer mRows.Close()
	for mRows.Next() {
		var tc models.TimeCount
		if err := mRows.Scan(&tc.Period, &tc.Count); err != nil {
			return d, err
		}
		d.KunjunganBulanan = append(d.KunjunganBulanan, tc)
	}

	// 13. Rata-rata Waktu Kunjungan per Poli
	timeQ := "SELECT a.id_poli, AVG(TIMESTAMPDIFF(SECOND,a.created_at,b.created_at))/60 AS avg_duration FROM Antrian a JOIN Billing b ON b.id_antrian=a.id_antrian WHERE a.created_at BETWEEN ? AND ?"
	tArgs := []interface{}{start, end}
	if idPoli != nil {
		timeQ += " AND a.id_poli = ?"
		tArgs = append(tArgs, *idPoli)
	}
	timeQ += " GROUP BY a.id_poli"
	tRows, err := svc.DB.Query(timeQ, tArgs...)
	if err != nil {
		return d, err
	}
	defer tRows.Close()
	for tRows.Next() {
		var pd models.PoliDuration
		if err := tRows.Scan(&pd.IDPoli, &pd.AvgDuration); err != nil {
			return d, err
		}
		d.WaktuKunjunganAvg = append(d.WaktuKunjunganAvg, pd)
	}

	return d, nil
}

// helper to get *int
func ptrInt(i int) *int { return &i }

