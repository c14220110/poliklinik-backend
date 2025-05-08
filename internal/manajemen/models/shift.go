package models

type ShiftAssignItem struct {
	IDKaryawan int      `json:"id_karyawan"`
	NamaRole   []string `json:"nama_role"`      // ["Dokter","Suster"]
	JamMulai   string   `json:"jam_mulai"`      // "HH:MM:SS"
	JamAkhir   string   `json:"jam_akhir"`      // "HH:MM:SS"
}
