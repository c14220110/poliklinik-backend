package models

type BillingItem struct {
	Tindakan string `json:"tindakan_yang_diambil"` // id_icd9_cm
	Jumlah   int    `json:"jumlah"`
}

type InputBillingRequest struct {
	Tindakan        []BillingItem `json:"tindakan"`
	NamaPICTindakan int           `json:"nama_pic_tindakan"` // id_karyawan PIC
	Tanggal         string        `json:"tanggal"`           // opsional; tdk disimpan
}

//detail bilin
type DetailBilling struct {
	IDKunjungan           int               `json:"id_kunjungan"`
	NamaPasien             string           `json:"nama_pasien"`
	IDRM                   string           `json:"id_rm"`
	NamaPoli               string           `json:"nama_poli"`
	NamaDokter             string           `json:"nama_dokter"`
	BiayaDokter            float64          `json:"biaya_dokter"`
	KaryawanYangDitugaskan string           `json:"karyawan_yang_ditugaskan"`
	NamaAdministrasi       string           `json:"nama_administrasi"`
	Obat                   []ObatDetail     `json:"obat"`
	Tindakan               []TindakanDetail `json:"tindakan"`
	WaktuDibayar           *string          `json:"waktu_dibayar"`
}

type ObatDetail struct {
	NamaObat     string            `json:"nama_obat,omitempty"`
	Keterangan   string            `json:"keterangan"`
	Jumlah       int               `json:"jumlah"`
	Satuan       string            `json:"satuan,omitempty"`
	HargaSatuan  float64           `json:"harga_satuan,omitempty"`
	HargaTotal   float64           `json:"harga_total"`
	Instruksi    string            `json:"instruksi"`
	NamaRacikan  string            `json:"nama_racikan,omitempty"`
	Kemasan      string            `json:"kemasan,omitempty"`
	Komposisi    []KomposisiDetail `json:"komposisi,omitempty"`
}

type KomposisiDetail struct {
	NamaObat    string  `json:"nama_obat"`
	Dosis       int     `json:"dosis"`
	Satuan      string  `json:"satuan"`
	HargaSatuan float64 `json:"harga_satuan"`
}

type TindakanDetail struct {
	NamaTindakan       string  `json:"nama_tindakan"`
	Jumlah             int     `json:"jumlah"`
	HargaTindakan      float64 `json:"harga_tindakan"`
	TotalHargaTindakan float64 `json:"total_harga_tindakan"`
}