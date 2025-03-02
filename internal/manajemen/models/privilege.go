package models

import "time"

type Privilege struct {
	IDPrivilege   int        `json:"id_privilege"`
	NamaPrivilege string     `json:"nama_privilege"`
	Deskripsi     string     `json:"deskripsi"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}
