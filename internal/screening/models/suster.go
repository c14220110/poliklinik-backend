package models

import "time"

// Suster mewakili data Karyawan yang memiliki peran Suster.
type Suster struct {
	ID_Suster int       `json:"id_suster" db:"ID_Karyawan"`
	Nama      string    `json:"nama" db:"Nama"`
	Username  string    `json:"username" db:"Username"`
	Password  string    `json:"password" db:"Password"`
	CreatedAt time.Time `json:"created_at" db:"Created_At"`
}
