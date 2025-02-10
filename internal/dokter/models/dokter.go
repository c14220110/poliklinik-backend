package models

import "time"

// Dokter mewakili data Karyawan yang memiliki peran Dokter.
type Dokter struct {
	ID_Dokter int       `json:"id_dokter" db:"ID_Karyawan"` // gunakan ID_Karyawan sebagai ID_Dokter
	Nama      string    `json:"nama" db:"Nama"`
	Username  string    `json:"username" db:"Username"`
	Password  string    `json:"password" db:"Password"`
	// Spesialisasi bisa ditentukan dari Detail_Role_Karyawan atau di-set default
	Spesialisasi string    `json:"spesialisasi" db:"Spesialisasi"`
	CreatedAt    time.Time `json:"created_at" db:"Created_At"`
}
