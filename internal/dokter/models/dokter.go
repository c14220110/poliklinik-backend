package models

import "time"

// Dokter merepresentasikan data dokter pada sistem.
// Field Password biasanya tidak dikirim dalam response (disembunyikan dengan tag JSON "-").
type Dokter struct {
	ID_Dokter  int        `json:"id_dokter"`
	Nama       string     `json:"nama"`
	Username   string     `json:"username"`
	Password   string     `json:"-"` // tidak akan di-encode dalam JSON
	ID_Role    int        `json:"id_role"`
	Privileges []int      `json:"privileges"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}
