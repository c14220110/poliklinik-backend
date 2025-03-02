package models

import "time"

// Suster mewakili data user suster yang digunakan untuk modul screening.
type Suster struct {
	ID_Suster  int       `json:"id_suster"`   // Mengacu pada ID_Karyawan di tabel Karyawan
	Nama       string    `json:"nama"`
	Username   string    `json:"username"`
	Password   string    `json:"-"`           // Tidak diekspos di JSON
	ID_Role    int       `json:"id_role"`     // Role yang dimiliki, harus sesuai dengan "Suster"
	Privileges []int     `json:"privileges"`  // Daftar id_privilege yang dimiliki
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}
