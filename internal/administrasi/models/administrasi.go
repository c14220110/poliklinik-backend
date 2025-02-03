package models

import "time"

// Administrasi mewakili data user administrasi.
type Administrasi struct {
	ID        int       `json:"id" db:"ID_Admin"`
	Nama      string    `json:"nama" db:"Nama"`
	Username  string    `json:"username" db:"Username"`
	Password  string    `json:"-" db:"Password"` // jangan expose password
	CreatedAt time.Time `json:"created_at" db:"Created_At"`
}
