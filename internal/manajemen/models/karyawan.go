package models

import "time"

// Karyawan adalah representasi data karyawan yang ada di tabel Karyawan.
type Karyawan struct {
	IDKaryawan   int64     `json:"id_karyawan"` // Ubah tipe ke int64
	NIK          string    `json:"nik"`
	Nama         string    `json:"nama"`
	Username     string    `json:"username"`
	Password     string    `json:"password"`
	TanggalLahir time.Time `json:"tanggal_lahir"`
	Alamat       string    `json:"alamat"`
	NoTelp       string    `json:"no_telp"`
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedAt    time.Time `json:"created_at"`
	DeletedAt    time.Time `json:"deleted_at,omitempty"`
}
