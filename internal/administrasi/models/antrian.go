package models

import "time"

// Antrian mewakili data antrian pasien.
type Antrian struct {
	ID           int       `json:"id" db:"ID_Antrian"`
	IDPasien     int       `json:"id_pasien" db:"ID_Pasien"`
	IDPoli       int       `json:"id_poli" db:"ID_Poli"`
	NomorAntrian int       `json:"nomor_antrian" db:"Nomor_Antrian"`
	Status       int       `json:"status" db:"Status"` // misal: 0=menunggu, 1=proses, 2=selesai
	CreatedAt    time.Time `json:"created_at" db:"Created_At"`
}
