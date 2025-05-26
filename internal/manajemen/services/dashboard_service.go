package services

import (
	"database/sql"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
)

type DashboardService struct {
	DB *sql.DB
}

func NewDashboardService(db *sql.DB) *DashboardService {
	return &DashboardService{DB: db}
}


// GetDashboardData menyusun seluruh metrik dashboard.
func (svc *DashboardService) GetDashboardData(
	idPoli *int, start, end time.Time,
) (models.DashboardData, error) {

	var d models.DashboardData
	// perpanjang rentang akhir ke 23:59:59
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	/* ------------------------------------------------------------------
	   1-4.  Hitung pasien (dibatalkan / konsultasi / menunggu / total)
	   ------------------------------------------------------------------ */
	countAntrian := func(status *int) (int, error) {
		var (
			q   string
			arg []interface{}
		)
		if status != nil {
			q = `SELECT COUNT(*) FROM Antrian
			     WHERE id_status = ? AND updated_at BETWEEN ? AND ?`
			arg = []interface{}{*status, start, end}
		} else {
			q = `SELECT COUNT(*) FROM Antrian
			     WHERE created_at BETWEEN ? AND ?`
			arg = []interface{}{start, end}
		}
		if idPoli != nil && status != nil { // untuk by-status pakai updated_at
			q += ` AND id_poli = ?`
			arg = append(arg, *idPoli)
		}
		var cnt int
		if err := svc.DB.QueryRow(q, arg...).Scan(&cnt); err != nil {
			return 0, err
		}
		return cnt, nil
	}

	var err error
	if d.PasienDibatalkan, err = countAntrian(ptrInt(7)); err != nil {
		return d, err
	}
	if d.PasienKonsultasi, err = countAntrian(ptrInt(5)); err != nil {
		return d, err
	}
	if d.PasienMenunggu, err = countAntrian(ptrInt(1)); err != nil {
		return d, err
	}
	if d.TotalPasien, err = countAntrian(nil); err != nil {
		return d, err
	}

	/* ------------------------------------------------------------------
	   5.  Karyawan aktif / non-aktif
	   ------------------------------------------------------------------ */
	if err = svc.DB.QueryRow(`SELECT COUNT(*) FROM Karyawan WHERE deleted_at IS NULL`).
		Scan(&d.KaryawanAktif); err != nil {
		return d, err
	}
	if err = svc.DB.QueryRow(`SELECT COUNT(*) FROM Karyawan WHERE deleted_at IS NOT NULL`).
		Scan(&d.KaryawanNonAktif); err != nil {
		return d, err
	}

	/* ------------------------------------------------------------------
	   6.  Tren penyakit (ICD-10)
	   ------------------------------------------------------------------ */
	icdQ := `SELECT i.display, COUNT(*)
	         FROM Assessment a JOIN ICD10 i ON a.id_icd10 = i.id_icd10
	         WHERE a.created_at BETWEEN ? AND ?`
	args := []interface{}{start, end}
	if idPoli != nil {
		icdQ += ` AND a.id_poli = ?`
		args = append(args, *idPoli)
	}
	icdQ += ` GROUP BY i.display`
	rows, err := svc.DB.Query(icdQ, args...)
	if err != nil { return d, err }
	defer rows.Close()

	for rows.Next() {
		var t models.PenyakitTrend
		if err = rows.Scan(&t.Display, &t.Count); err != nil { return d, err }
		d.TrenPenyakit = append(d.TrenPenyakit, t)
	}

	/* ------------------------------------------------------------------
	   7-8.  Pendapatan total & rata-rata
	   ------------------------------------------------------------------ */
	billQ := `SELECT COALESCE(SUM(b.total),0)
	          FROM Billing b JOIN Antrian a ON b.id_antrian = a.id_antrian
	          WHERE b.created_at BETWEEN ? AND ?`
	bArgs := []interface{}{start, end}
	if idPoli != nil {
		billQ += ` AND a.id_poli = ?`
		bArgs = append(bArgs, *idPoli)
	}
	if err = svc.DB.QueryRow(billQ, bArgs...).Scan(&d.PendapatanTotal); err != nil {
		return d, err
	}
	days := end.Sub(start).Hours()/24 + 1
	if days > 0 {
		d.PendapatanRataRata = d.PendapatanTotal / days
	}

	/* ------------------------------------------------------------------
	   9.  Kunjungan terbanyak  (SELALU lintas poli, abaikan id_poli filter)
	   ------------------------------------------------------------------ */
	kmRows, err := svc.DB.Query(`
		SELECT id_poli, COUNT(*)
		FROM   Antrian
		WHERE  created_at BETWEEN ? AND ?
		GROUP  BY id_poli
		ORDER  BY COUNT(*) DESC`,
		start, end)
	if err != nil { return d, err }
	defer kmRows.Close()

	for kmRows.Next() {
		var pc models.PoliCount
		if err = kmRows.Scan(&pc.IDPoli, &pc.Count); err != nil { return d, err }
		d.KunjunganTerbanyak = append(d.KunjunganTerbanyak, pc)
	}

	/* ------------------------------------------------------------------
	   10-12.  Kunjungan harian / mingguan / bulanan (masih hormati id_poli)
	   ------------------------------------------------------------------ */
	if err = svc.fillTimeSeries(&d, idPoli, start, end); err != nil {
		return d, err
	}

	/* ------------------------------------------------------------------
	   13.  Durasi rata-rata pasien selesai (Billing.status = 2)
	   ------------------------------------------------------------------ */
	durQ := `SELECT AVG(TIMESTAMPDIFF(SECOND, a.created_at, b.updated_at))/60
	         FROM Antrian a
	         JOIN Billing b ON b.id_antrian = a.id_antrian
	         WHERE b.id_status = 2
	           AND a.created_at BETWEEN ? AND ?
	           AND b.updated_at BETWEEN ? AND ?`
	dArgs := []interface{}{start, end, start, end}
	if idPoli != nil {
		durQ += ` AND a.id_poli = ?`
		dArgs = append(dArgs, *idPoli)
	}
	if err = svc.DB.QueryRow(durQ, dArgs...).Scan(&d.DurasiKunjungan); err != nil {
		return d, err
	}

	return d, nil
}

/* ------------------------------------------------------------------ */
/* ------------------------- helper func ---------------------------  */
/* ------------------------------------------------------------------ */

func (svc *DashboardService) fillTimeSeries(
	d *models.DashboardData, idPoli *int, start, end time.Time,
) error {

	type row struct{ period string; count int }
	exec := func(q string, args []interface{}, dest *[]models.TimeCount) error {
		r, err := svc.DB.Query(q, args...)
		if err != nil { return err }
		defer r.Close()
		for r.Next() {
			var x row
			if err := r.Scan(&x.period, &x.count); err != nil { return err }
// SESUDAH – keyed fields, warning hilang
*dest = append(*dest, models.TimeCount{
	Period: x.period,
	Count:  x.count,
})		}
		return nil
	}

	// harian
	hQ := `SELECT DAYNAME(created_at), COUNT(*) FROM Antrian
	       WHERE created_at BETWEEN ? AND ?`
	hArgs := []interface{}{start, end}
	// mingguan
	wQ := `SELECT YEARWEEK(created_at,1), COUNT(*) FROM Antrian
	       WHERE created_at BETWEEN ? AND ?`
	wArgs := []interface{}{start, end}
	// bulanan
	mQ := `SELECT DATE_FORMAT(created_at,'%Y-%m'), COUNT(*) FROM Antrian
	       WHERE created_at BETWEEN ? AND ?`
	mArgs := []interface{}{start, end}

	if idPoli != nil {
		hQ += ` AND id_poli = ?`
		wQ += ` AND id_poli = ?`
		mQ += ` AND id_poli = ?`
		hArgs, wArgs, mArgs =
			append(hArgs, *idPoli), append(wArgs, *idPoli), append(mArgs, *idPoli)
	}

	hQ += ` GROUP BY DAYNAME(created_at)`
	wQ += ` GROUP BY YEARWEEK(created_at,1)`
	mQ += ` GROUP BY DATE_FORMAT(created_at,'%Y-%m') ORDER BY 1`

	if err := exec(hQ, hArgs, &d.KunjunganHarian); err != nil { return err }
	if err := exec(wQ, wArgs, &d.KunjunganMingguan); err != nil { return err }
	if err := exec(mQ, mArgs, &d.KunjunganBulanan); err != nil { return err }
	return nil
}

func ptrInt(i int) *int { return &i }
