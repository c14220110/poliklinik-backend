package models

type ShiftKaryawan struct {
    ID_Shift_Karyawan int    `json:"id_shift_karyawan"`
    ID_Karyawan       int    `json:"id_karyawan"`
    ID_Poli           int    `json:"id_poli"`
    CustomJamMulai    string `json:"custom_jam_mulai"`   // Format: "15:04:05"
    CustomJamSelesai  string `json:"custom_jam_selesai"` // Format: "15:04:05"
    Tanggal           string `json:"tanggal"`            // Format: "2006-01-02"
}
