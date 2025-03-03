package models

import "time"

type Pasien struct {
	IDPasien      int       `json:"id_pasien"`
	Nama          string    `json:"nama"`
	TanggalLahir  time.Time `json:"tanggal_lahir"`
	JenisKelamin  string    `json:"jenis_kelamin"`
	TempatLahir   string    `json:"tempat_lahir"`
	NIK           string    `json:"nik"`
	Kelurahan     string    `json:"kelurahan"`
	Kecamatan     string    `json:"kecamatan"`
	KotaTinggal   string    `json:"kota_tinggal"`
	Alamat        string    `json:"alamat"`
	NoTelp        string    `json:"no_telp"`
}
