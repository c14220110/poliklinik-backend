package models

import "time"

// Screening mewakili record di tabel Screening.
type Screening struct {
	ID_Screening   int       `json:"id_screening"`
	ID_Pasien      int       `json:"id_pasien"`
	ID_Karyawan    int       `json:"id_karyawan"` // ID operator (suster) yang melakukan screening
	Systolic       int       `json:"systolic"`
	Diastolic      int       `json:"diastolic"`
	Berat_Badan    float64   `json:"berat_badan"`
	Suhu_Tubuh     float64   `json:"suhu_tubuh"`
	Tinggi_Badan   float64   `json:"tinggi_badan"`
	Detak_Nadi     int       `json:"detak_nadi"`
	Laju_Respirasi int       `json:"laju_respirasi"`
	Keterangan     string    `json:"keterangan"`
	Created_At     time.Time `json:"created_at"`
}
