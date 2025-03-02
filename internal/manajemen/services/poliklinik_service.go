package services

import (
	"database/sql"
	"fmt"
	"strings"
)

type PoliklinikService struct {
	DB *sql.DB
}

func NewPoliklinikService(db *sql.DB) *PoliklinikService {
	return &PoliklinikService{DB: db}
}

// GetPoliklinikListFiltered mengambil data dari tabel Poliklinik dengan hanya menampilkan kolom yang diperlukan.
// Jika statusFilter tidak kosong, maka:
//   - status=aktif akan memfilter dengan id_status = 1
//   - status=nonaktif akan memfilter dengan id_status = 2
func (ps *PoliklinikService) GetPoliklinikListFiltered(statusFilter string) ([]map[string]interface{}, error) {
	baseQuery := `
		SELECT id_poli, id_status, nama_poli, jumlah_tenkes, logo_poli, keterangan
		FROM Poliklinik
	`
	conditions := []string{}
	params := []interface{}{}

	if statusFilter != "" {
		s := strings.ToLower(statusFilter)
		if s == "aktif" {
			conditions = append(conditions, "id_status = ?")
			params = append(params, 1)
		} else if s == "nonaktif" {
			conditions = append(conditions, "id_status = ?")
			params = append(params, 0)
		}
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY id_poli"

	rows, err := ps.DB.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var idPoli, idStatus, jumlahTenkes int
		var namaPoli, keterangan string
		var logoPoli sql.NullString

		if err := rows.Scan(&idPoli, &idStatus, &namaPoli, &jumlahTenkes, &logoPoli, &keterangan); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}

		record := map[string]interface{}{
			"id_poli":       idPoli,
			"id_status":     idStatus,
			"nama_poli":     namaPoli,
			"jumlah_tenkes": jumlahTenkes,
			"logo_poli":     nil,
			"keterangan":    keterangan,
		}
		if logoPoli.Valid {
			record["logo_poli"] = logoPoli.String
		}
		list = append(list, record)
	}

	return list, nil
}

// SoftDeletePoliklinik melakukan soft delete dengan mengupdate kolom deleted_at,
// mengubah id_status menjadi 0 (nonaktif) dan mencatat deleted_by di tabel Management_Poli.
func (ps *PoliklinikService) SoftDeletePoliklinik(idPoli int, idManagement int) error {
	tx, err := ps.DB.Begin()
	if err != nil {
		return err
	}

	// Pastikan rollback jika terjadi error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Update tabel Poliklinik: set deleted_at = NOW(), id_status = 0 (nonaktif)
	queryPoli := `UPDATE Poliklinik SET deleted_at = NOW(), id_status = 0 WHERE id_poli = ?`
	res, err := tx.Exec(queryPoli, idPoli)
	if err != nil {
		return fmt.Errorf("gagal mengupdate Poliklinik: %v", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal mendapatkan affected rows: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tidak ditemukan Poliklinik dengan id %d", idPoli)
	}

	// Update tabel Management_Poli: set deleted_by = idManagement untuk record dengan id_poli tersebut.
	// Pastikan tabel Management_Poli memiliki kolom deleted_by.
	queryMgmtPoli := `UPDATE Management_Poli SET deleted_by = ? WHERE id_poli = ?`
	res, err = tx.Exec(queryMgmtPoli, idManagement, idPoli)
	if err != nil {
		return fmt.Errorf("gagal mengupdate Management_Poli: %v", err)
	}
	rowsAffected, err = res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal mendapatkan affected rows untuk Management_Poli: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tidak ditemukan record Management_Poli untuk id_poli %d", idPoli)
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// AddPoliklinikWithManagement menambahkan record Poliklinik dan mencatat siapa yang menambahkannya ke tabel Management_Poli.
// Untuk Poliklinik baru, id_status diset 1 (aktif).
func (ps *PoliklinikService) AddPoliklinikWithManagement(namaPoli, keterangan, logoPath string, idManagement int) (int, error) {
	tx, err := ps.DB.Begin()
	if err != nil {
		return 0, err
	}

	// Pastikan rollback jika terjadi error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Insert ke tabel Poliklinik dengan id_status = 1 (aktif)
	queryPoli := `INSERT INTO Poliklinik (nama_poli, keterangan, logo_poli, id_status) VALUES (?, ?, ?, 1)`
	res, err := tx.Exec(queryPoli, namaPoli, keterangan, logoPath)
	if err != nil {
		return 0, fmt.Errorf("failed to insert Poliklinik: %v", err)
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get inserted ID: %v", err)
	}
	poliID := int(lastID)

	// Catat ke tabel Management_Poli
	// Asumsikan tabel Management_Poli memiliki kolom: id_management, id_poli, created_by, updated_by, deleted_by.
	queryMgmt := `INSERT INTO Management_Poli (id_management, id_poli, created_by, updated_by, deleted_by) VALUES (?, ?, ?, ?, NULL)`
	_, err = tx.Exec(queryMgmt, idManagement, poliID, idManagement, idManagement)
	if err != nil {
		return 0, fmt.Errorf("failed to record Management_Poli: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return poliID, nil
}

// UpdatePoliklinikWithOptionalLogo mengupdate data Poliklinik berdasarkan id_poli.
// Jika logoPoli tidak kosong, maka kolom logo_poli diupdate; jika kosong, logo_poli tidak diubah.
// Selain itu, mencatat siapa yang melakukan update di tabel Management_Poli dengan mengupdate kolom updated_by.
func (ps *PoliklinikService) UpdatePoliklinikWithOptionalLogo(idPoli int, namaPoli, keterangan, logoPoli string, idManagement int) error {
	tx, err := ps.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var query string
	var args []interface{}

	// Jika file logo diupload, update kolom logo_poli juga.
	if logoPoli != "" {
		query = `UPDATE Poliklinik SET nama_poli = ?, keterangan = ?, logo_poli = ?, updated_at = NOW() WHERE id_poli = ?`
		args = []interface{}{namaPoli, keterangan, logoPoli, idPoli}
	} else {
		// Jika tidak, update hanya nama_poli dan keterangan.
		query = `UPDATE Poliklinik SET nama_poli = ?, keterangan = ?, updated_at = NOW() WHERE id_poli = ?`
		args = []interface{}{namaPoli, keterangan, idPoli}
	}

	res, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update Poliklinik: %v", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("poliklinik with id %d not found", idPoli)
	}

	// Catat update di tabel Management_Poli: set updated_by = idManagement
	queryMgmt := `UPDATE Management_Poli SET updated_by = ? WHERE id_poli = ?`
	res, err = tx.Exec(queryMgmt, idManagement, idPoli)
	if err != nil {
		return fmt.Errorf("failed to update Management_Poli: %v", err)
	}
	_, err = res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows for Management_Poli: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}