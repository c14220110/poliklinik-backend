package models

type BillingItem struct {
	Tindakan string  `json:"tindakan_yang_diambil"` // id_icd9_cm
	Jumlah   int     `json:"jumlah"`
	Harga    float64 `json:"harga_tindakan"`
}

type InputBillingRequest struct {
	Tindakan       []BillingItem `json:"tindakan"`
	NamaPICTindakan int          `json:"nama_pic_tindakan"` // id_karyawan PIC
	Tanggal        string        `json:"tanggal"`           // opsional; tdk disimpan
}