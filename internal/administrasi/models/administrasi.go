package models

import "time"

// Administrasi mewakili data admin yang diambil dari tabel Karyawan.
type Administrasi struct {
	ID_Admin  int       `json:"id" db:"ID_Admin"` // Akan menyimpan ID_Karyawan
	Nama      string    `json:"nama" db:"Nama"`
	Username  string    `json:"username" db:"Username"`
	Password  string    `json:"-" db:"Password"`
	CreatedAt time.Time `json:"created_at" db:"Created_At"`
}
