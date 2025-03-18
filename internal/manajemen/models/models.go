package models

type AssignShiftRequest struct {
    IDKaryawan int    `json:"id_karyawan"`
    IDRole     int    `json:"id_role"`
    JamMulai   string `json:"jam_mulai"`
    JamAkhir   string `json:"jam_akhir"`
}