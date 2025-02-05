package models

import "time"

// ShiftSuster mewakili data shift suster.
type ShiftSuster struct {
	ID_Shift     int       `json:"id_shift" db:"ID_Shift"`
	ID_Suster    int       `json:"id_suster" db:"ID_Suster"`
	ID_Poli      int       `json:"id_poli" db:"ID_Poli"`
	ID_Management int      `json:"id_management" db:"ID_Management"`
	Jam_Mulai    time.Time `json:"jam_mulai" db:"Jam_Mulai"`
	Jam_Selesai  time.Time `json:"jam_selesai" db:"Jam_Selesai"`
}
