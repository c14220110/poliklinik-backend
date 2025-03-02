package models

import "time"

type Administrasi struct {
	ID_Admin    int       `json:"id_karyawan"` // menggunakan id_karyawan sebagai ID_Admin
	Nama        string    `json:"nama"`
	Username    string    `json:"username"`
	Password    string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	ID_Role     int       `json:"id_role"`      // role yang dimiliki
	Privileges  []int     `json:"privileges"`   // daftar id_privilege
}
