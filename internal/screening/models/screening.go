package models

import "time"

// Screening merepresentasikan data screening yang disimpan di database
type Screening struct {
	ID_Screening   int       `json:"id_screening"`
	ID_Pasien      int       `json:"id_pasien"`
	ID_Karyawan    int       `json:"id_karyawan"`
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

// ScreeningInput merepresentasikan data input screening dari request
type ScreeningInput struct {
	Systolic       int     `json:"systolic"`
	Diastolic      int     `json:"diastolic"`
	Berat_Badan    float64 `json:"berat_badan"`
	Suhu_Tubuh     float64 `json:"suhu_tubuh"`
	Tinggi_Badan   float64 `json:"tinggi_badan"`
	Detak_Nadi     int     `json:"detak_nadi"`
	Laju_Respirasi int     `json:"laju_respirasi"`
	Keterangan     string  `json:"keterangan"`
}