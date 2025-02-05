package models

import "time"

// Dokter mewakili data dokter pada tabel Dokter.
type Dokter struct {
	ID_Dokter    int       `json:"id_dokter" db:"ID_Dokter"`
	Nama         string    `json:"nama" db:"Nama"`
	Username     string    `json:"username" db:"Username"`
	Password     string    `json:"password" db:"Password"`
	Spesialisasi string    `json:"spesialisasi" db:"Spesialisasi"`
	CreatedAt    time.Time `json:"created_at" db:"Created_At"`
}
