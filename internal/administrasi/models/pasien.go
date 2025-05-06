package models

import "time"


type Pasien struct {
	IDPasien         int       `json:"id_pasien"`
	Nama             string    `json:"nama"`
	TanggalLahir     time.Time `json:"tanggal_lahir"`
	JenisKelamin     string    `json:"jenis_kelamin"`
	TempatLahir      string    `json:"tempat_lahir"`
	NIK              string    `json:"nik"`
	Kelurahan        string    `json:"kelurahan"`
	Kecamatan        string    `json:"kecamatan"`
	KotaTinggal      string    `json:"kota_tinggal"`
	Alamat           string    `json:"alamat"`
	NoTelp           string    `json:"no_telp"`
	IDAgama          int       `json:"id_agama"`
	StatusPerkawinan int       `json:"status_perkawinan"`
	Pekerjaan        string    `json:"pekerjaan"`
}

type ExtendedPasienRequest struct {
	Nama              string `json:"nama"`
	TanggalLahir      string `json:"tanggal_lahir"`
	JenisKelamin      string `json:"jenis_kelamin"`
	TempatLahir       string `json:"tempat_lahir"`
	Nik               string `json:"nik"`
	Kelurahan         string `json:"kelurahan"`
	Kecamatan         string `json:"kecamatan"`
	KotaTempatTinggal string `json:"kota_tempat_tinggal"`
	Alamat            string `json:"alamat"`
	NoTelp            string `json:"no_telp"`
	IDPoli            int    `json:"id_poli"`
	KeluhanUtama      string `json:"keluhan_utama"`
	Agama             string `json:"agama"`
	StatusPerkawinan  string `json:"status_perkawinan"`
	Pekerjaan         string `json:"pekerjaan"`
	PenanggungJawab   string `json:"penanggung_jawab"`
}