package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
)

type DashboardService struct {
	DB *sql.DB
}

func NewDashboardService(db *sql.DB) *DashboardService {
	return &DashboardService{DB: db}
}

// GetDashboardData menghimpun semua metrik dashboard.
func (svc *DashboardService) GetDashboardData(
	idPoli *int, start, end time.Time,
) (models.DashboardData, error) {

	var d models.DashboardData
	// sesuaikan end → jam 23:59:59 hari yg sama
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	/* ---------- helper count by status / total ---------- */
	countQuery := func(status *int) (int, error) {
		var (
			q      string
			params []interface{}
		)
		if status != nil {
			q = `SELECT COUNT(*) FROM Antrian
			     WHERE id_status = ? AND updated_at BETWEEN ? AND ?`
			params = []interface{}{*status, start, end}
		} else {
			q = `SELECT COUNT(*) FROM Antrian
			     WHERE created_at BETWEEN ? AND ?`
			params = []interface{}{start, end}
		}
		if idPoli != nil {
			q += " AND id_poli = ?"
			params = append(params, *idPoli)
		}
		var cnt int
		if err := svc.DB.QueryRow(q, params...).Scan(&cnt); err != nil {
			return 0, err
		}
		return cnt, nil
	}

	/* ---------- 1-4 Pasien ---------- */
	var err error
	if d.PasienDibatalkan, err = countQuery(ptrInt(7)); err != nil {
		return d, err
	}
	if d.PasienKonsultasi, err = countQuery(ptrInt(5)); err != nil {
		return d, err
	}
	if d.PasienMenunggu, err = countQuery(ptrInt(1)); err != nil {
		return d, err
	}
	if d.TotalPasien, err = countQuery(nil); err != nil {
		return d, err
	}

	/* ---------- 5 Tenaga Kesehatan ---------- */
	if err = svc.DB.QueryRow(`SELECT COUNT(*) FROM Karyawan WHERE deleted_at IS NULL`).
		Scan(&d.KaryawanAktif); err != nil {
		return d, err
	}
	if err = svc.DB.QueryRow(`SELECT COUNT(*) FROM Karyawan WHERE deleted_at IS NOT NULL`).
		Scan(&d.KaryawanNonAktif); err != nil {
		return d, err
	}

	/* ---------- 6 Tren Penyakit ---------- */
	trendQ := `
		SELECT i.display, COUNT(*)
		FROM   Assessment a
		       JOIN ICD10 i ON a.id_icd10 = i.id_icd10
		WHERE  a.created_at BETWEEN ? AND ?`
	args := []interface{}{start, end}
	if idPoli != nil {
		trendQ += " AND a.id_poli = ?"
		args = append(args, *idPoli)
	}
	trendQ += " GROUP BY i.display"
	rows, err := svc.DB.Query(trendQ, args...)
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

	/* ---------- 7-8 Pendapatan ---------- */
	billQ := `
		SELECT COALESCE(SUM(b.total),0)
		FROM   Billing b
		       JOIN Antrian a ON b.id_antrian = a.id_antrian
		WHERE  b.created_at BETWEEN ? AND ?`
	billArgs := []interface{}{start, end}
	if idPoli != nil {
		billQ += " AND a.id_poli = ?"
		billArgs = append(billArgs, *idPoli)
	}
	if err := svc.DB.QueryRow(billQ, billArgs...).Scan(&d.PendapatanTotal); err != nil {
		return d, err
	}
	days := end.Sub(start).Hours()/24 + 1
	if days > 0 {
		d.PendapatanRataRata = d.PendapatanTotal / days
	}

	/* ---------- 9 Kunjungan Terbanyak (abaikan id_poli filter) ---------- */
	kunjQ := `
		SELECT a.id_poli, COUNT(*)
		FROM   Antrian a
		WHERE  a.created_at BETWEEN ? AND ?
		GROUP  BY a.id_poli
		ORDER  BY COUNT(*) DESC`
	kRows, err := svc.DB.Query(kunjQ, start, end)
	if err != nil {
		return d, err
	}
	defer kRows.Close()
	for kRows.Next() {
		var pc models.PoliCount
		if err := kRows.Scan(&pc.IDPoli, &pc.Count); err != nil {
			return d, err
		}
		d.KunjunganTerbanyak = append(d.KunjunganTerbanyak, pc)
	}

	/* ---------- 10-12 Kunjungan Harian / Mingguan / Bulanan (masih hormati id_poli) ---------- */
	timeAgg := []struct {
		dest *[]models.TimeCount
		sql  string
	}{
		{&d.KunjunganHarian,
			`SELECT DAYNAME(created_at), COUNT(*) FROM Antrian
			 WHERE created_at BETWEEN ? AND ?%s GROUP BY DAYNAME(created_at)`},
		{&d.KunjunganMingguan,
			`SELECT YEARWEEK(created_at,1), COUNT(*) FROM Antrian
			 WHERE created_at BETWEEN ? AND ?%s GROUP BY YEARWEEK(created_at,1)`},
		{&d.KunjunganBulanan,
			`SELECT DATE_FORMAT(created_at,'%Y-%m'), COUNT(*) FROM Antrian
			 WHERE created_at BETWEEN ? AND ?%s GROUP BY DATE_FORMAT(created_at,'%Y-%m') ORDER BY 1`},
	}
	for _, tg := range timeAgg {
		cond := ""
		params := []interface{}{start, end}
		if idPoli != nil {
			cond = " AND id_poli = ?"
			params = append(params, *idPoli)
		}
		q := fmt.Sprintf(tg.sql, cond)
		r, err := svc.DB.Query(q, params...)
		if err != nil {
			return d, err
		}
		for r.Next() {
			var tc models.TimeCount
			if err := r.Scan(&tc.Period, &tc.Count); err != nil {
				r.Close(); return d, err
			}
			*tg.dest = append(*tg.dest, tc)
		}
		r.Close()
	}

	/* ---------- 13 Durasi Pasien per Kunjungan ---------- */
	durQ := `
		SELECT AVG(TIMESTAMPDIFF(SECOND, a.created_at, b.updated_at))/60
		FROM   Billing b
		       JOIN Antrian a ON a.id_antrian = b.id_antrian
		WHERE  b.id_status = 2
		       AND b.updated_at BETWEEN ? AND ?`
	durArgs := []interface{}{start, end}
	if idPoli != nil {
		durQ += " AND a.id_poli = ?"
		durArgs = append(durArgs, *idPoli)
	}
	if err := svc.DB.QueryRow(durQ, durArgs...).Scan(&d.DurasiPasienPerKunjungan); err != nil {
		return d, err
	}

	return d, nil
}

/* ------------- util ------------- */
func ptrInt(i int) *int { return &i }