package models

import "time"

type Role struct {
	IDRole    int       `json:"id_role"`
	NamaRole  string    `json:"nama_role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt time.Time `json:"deleted_at,omitempty"`
}
