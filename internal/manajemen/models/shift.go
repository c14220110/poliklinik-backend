package models

type ShiftAssignItem struct {
	IDKaryawan int      `json:"id_karyawan"`
	NamaRole   []string `json:"nama_role"`      // ["Dokter","Suster"]
	JamMulai   string   `json:"jam_mulai"`      // "HH:MM:SS"
	JamAkhir   string   `json:"jam_akhir"`      // "HH:MM:SS"
}
type ShiftDetail struct {
	IDShiftKaryawan int    `json:"id_shift_karyawan"`
	JamMulai        string `json:"custom_jam_mulai"`  // "15:00:00"
	JamAkhir        string `json:"custom_jam_akhir"`  // "21:00:00"
	NamaPoli        string `json:"nama_poli"`
}

type JadwalShiftPerHari struct {
	Tanggal   string        `json:"tanggal"`   // "08/06/2025"
	Hari      string        `json:"hari"`      // "Senin"
	ShiftPagi []ShiftDetail `json:"shift_pagi"`
	ShiftSore []ShiftDetail `json:"shift_sore"`
}
