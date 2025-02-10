package models

import "time"

// Management mewakili data manajemen dari tabel Management.
type Management struct {
	ID_Management int       `json:"id_management" db:"ID_Management"`
	Username      string    `json:"username" db:"Username"`
	Password      string    `json:"password" db:"Password"`
	Nama          string    `json:"nama" db:"Nama"`
	CreatedAt     time.Time `json:"created_at" db:"Created_At"`
}
