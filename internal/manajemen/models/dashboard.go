package models

// DashboardData is the aggregated data for management dashboard
type DashboardData struct {
	PasienDibatalkan           int             `json:"pasien_dibatalkan"`
	PasienKonsultasi           int             `json:"pasien_konsultasi"`
	PasienMenunggu             int             `json:"pasien_menunggu"`
	TotalPasien                int             `json:"total_pasien"`
	KaryawanAktif              int             `json:"karyawan_aktif"`
	KaryawanNonAktif           int             `json:"karyawan_non_aktif"`
	TrenPenyakit               []PenyakitTrend `json:"tren_penyakit"`
	PendapatanTotal            float64         `json:"pendapatan_total"`
	PendapatanRataRata         float64         `json:"pendapatan_rata_rata"`
	KunjunganTerbanyak         []PoliCount     `json:"kunjungan_terbanyak"`
	KunjunganHarian            []TimeCount     `json:"kunjungan_harian"`
	KunjunganMingguan          []TimeCount     `json:"kunjungan_mingguan"`
	KunjunganBulanan           []TimeCount     `json:"kunjungan_bulanan"`
	DurasiPasienPerKunjungan   float64         `json:"durasi_pasien_per_kunjungan"` // menit
}

// --- struktur pendukung tak berubah ---
type PenyakitTrend struct {
	Display string `json:"display"`
	Count   int    `json:"count"`
}

type PoliCount struct {
	IDPoli int `json:"id_poli"`
	Count  int `json:"count"`
}

type TimeCount struct {
	Period string `json:"period"`
	Count  int    `json:"count"`
}
