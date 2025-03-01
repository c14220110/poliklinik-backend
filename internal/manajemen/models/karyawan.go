package models

import "time"

type Karyawan struct {
	IDKaryawan   int64     `json:"id_karyawan"`
	NIK          string    `json:"nik"`
	Nama         string    `json:"nama"`
	JenisKelamin string    `json:"jenis_kelamin"`
	Username     string    `json:"username"`
	Password     string    `json:"password"`
	TanggalLahir time.Time `json:"tanggal_lahir"`
	Alamat       string    `json:"alamat"`
	NoTelp       string    `json:"no_telp"`
	IDRole       int64     `json:"id_role"` // pastikan kolom ini ada untuk mengaitkan ke tabel Role
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedAt    time.Time `json:"created_at"`
	DeletedAt    time.Time `json:"deleted_at,omitempty"`
}
