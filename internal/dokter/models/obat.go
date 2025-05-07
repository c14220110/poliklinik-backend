package models

// Obat merepresentasikan record di tabel `Obat`
type Obat struct {
	IDObat      int     `json:"id_obat"      db:"id_obat"`
	Nama        string  `json:"nama"         db:"nama"`
	HargaSatuan float64 `json:"harga_satuan" db:"harga_satuan"`
	Satuan      string  `json:"satuan"       db:"satuan"`
	Jenis       string  `json:"jenis"        db:"jenis"`
	Stock       int     `json:"stock"        db:"stock"`
}
