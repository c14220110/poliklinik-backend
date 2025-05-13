package models

// Request utama
type ResepRequest struct {
    IDKunjungan int              `json:"id_kunjungan"`
    Sections    []SectionRequest `json:"sections"`
}

// Satu section resep
type SectionRequest struct {
    SectionType string               `json:"section_type"` // "obat" | "racikan"
    IDObat      *int                 `json:"id_obat,omitempty"`   // untuk obat tunggal
    NamaRacikan string               `json:"nama_racikan,omitempty"`
    Kemasan     string               `json:"kemasan,omitempty"`
    Jumlah      int                  `json:"jumlah"`
    Instruksi   string               `json:"instruksi"`
    Komposisi   []KomposisiRequest   `json:"komposisi,omitempty"` // jika racikan
}

type KomposisiRequest struct {
    IDObat int `json:"id_obat"`
    Dosis  int `json:"dosis"`
}