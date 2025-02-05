package models

import "time"

// ShiftDokter mewakili data shift untuk dokter.
type ShiftDokter struct {
	ID_Shift      int       `json:"id_shift" db:"ID_Shift"`
	ID_Dokter     int       `json:"id_dokter" db:"ID_Dokter"`
	ID_Poli       int       `json:"id_poli" db:"ID_Poli"`
	ID_Management int       `json:"id_management" db:"ID_Management"`
	Jam_Mulai     time.Time `json:"jam_mulai" db:"Jam_Mulai"`
	Jam_Selesai   time.Time `json:"jam_selesai" db:"Jam_Selesai"`
}
