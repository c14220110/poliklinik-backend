package models

import "time"

// Pasien mewakili data pasien.
type Pasien struct {
	ID                int       `json:"id" db:"ID_Pasien"`
	Nama              string    `json:"nama" db:"Nama"`
	TanggalLahir      time.Time `json:"tanggal_lahir" db:"Tanggal_Lahir"`
	JenisKelamin      string    `json:"jenis_kelamin" db:"Jenis_Kelamin"`
	TempatLahir       time.Time `json:"tempat_lahir" db:"Tempat_Lahir"` // Jika memang bertipe DATE
	Kelurahan         string    `json:"kelurahan" db:"Kelurahan"`
	Kecamatan         string    `json:"kecamatan" db:"Kecamatan"`
	Alamat            string    `json:"alamat,omitempty" db:"Alamat"`
	NoTelp            string    `json:"no_telp,omitempty" db:"No_Telp"`
	PoliTujuan        string    `json:"poli_tujuan" db:"Poli_Tujuan"`
	TanggalRegistrasi time.Time `json:"tanggal_registrasi" db:"Tanggal_Registrasi"`
}
