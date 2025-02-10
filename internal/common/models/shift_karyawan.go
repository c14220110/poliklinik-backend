package models

import "time"

// ShiftKaryawan mewakili record dari tabel Shift_Karyawan.
type ShiftKaryawan struct {
	ID_Shift    int       `json:"id_shift" db:"ID_Shift"`
	ID_Karyawan int       `json:"id_karyawan" db:"ID_Karyawan"`
	ID_Poli     int       `json:"id_poli" db:"ID_Poli"`
	Jam_Mulai   time.Time `json:"jam_mulai" db:"Jam_Mulai"`
	Jam_Selesai time.Time `json:"jam_selesai" db:"Jam_Selesai"`
}
