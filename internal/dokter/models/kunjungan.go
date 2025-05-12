package models

type RiwayatKunjungan struct {
    IDKunjungan   int      `json:"id_kunjungan"`
    Tanggal       string   `json:"tanggal"`
    TujuanPoli    string   `json:"tujuan_poli"`
    NomorAntrian  int      `json:"nomor_antrian"`
    KeluhanUtama  string   `json:"keluhan_utama"`
    HasilDiagnosa string   `json:"hasil_diagnosa"`
    Tindakan      []string `json:"tindakan"`
    IDResep       *int     `json:"id_resep"`
    IDAssessment  *int     `json:"id_assessment"`
}