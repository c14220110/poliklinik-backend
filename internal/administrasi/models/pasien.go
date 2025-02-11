package models

import "time"

// Pasien mewakili data pasien sesuai dengan tabel Pasien di database.
type Pasien struct {
	ID                int       `json:"id" db:"ID_Pasien"`
	Nama              string    `json:"nama" db:"Nama"`
	TanggalLahir      time.Time `json:"tanggal_lahir" db:"Tanggal_Lahir"`
	JenisKelamin      string    `json:"jenis_kelamin" db:"Jenis_Kelamin"`
	TempatLahir       string    `json:"tempat_lahir" db:"Tempat_Lahir"`
	NIK               string    `json:"nik" db:"NIK"`
	Kelurahan         string    `json:"kelurahan" db:"Kelurahan"`
	Kecamatan         string    `json:"kecamatan" db:"Kecamatan"`
	Alamat            string    `json:"alamat" db:"Alamat"`
	NoTelp            string    `json:"no_telp" db:"No_Telp"`
	TanggalRegistrasi time.Time `json:"tanggal_registrasi" db:"Tanggal_Registrasi"`
}
