package models

import "time"

// Billing mewakili data billing pasien.
type Billing struct {
	ID        int       `json:"id" db:"ID_Billing"`
	IDPasien  int       `json:"id_pasien" db:"ID_Pasien"`
	IDAdmin   int       `json:"id_admin" db:"ID_Admin"`
	Status    int       `json:"status" db:"Status"` // misal: 0=belum bayar, 1=sudah bayar
	CreatedAt time.Time `json:"created_at" db:"Created_At"`
}