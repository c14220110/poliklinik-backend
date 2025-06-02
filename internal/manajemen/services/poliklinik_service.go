package services

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type PoliklinikService struct {
	DB *sql.DB
}

func NewPoliklinikService(db *sql.DB) *PoliklinikService {
	return &PoliklinikService{DB: db}
}

// GetPoliklinikListFiltered mengambil daftar poliklinik dengan filter status (aktif / nonaktif / semua).
func (ps *PoliklinikService) GetPoliklinikListFiltered(statusFilter string) ([]map[string]interface{}, error) {
	baseQuery := `
		SELECT
			p.id_poli,
			p.id_status,
			p.nama_poli,
			p.jumlah_tenkes,
			p.logo_poli,
			p.keterangan,
			p.created_at
		FROM Poliklinik p
	`
	conditions := []string{}
	params := []interface{}{}

	if statusFilter != "" {
		switch strings.ToLower(statusFilter) {
		case "aktif":
			conditions = append(conditions, "p.id_status = ?")
			params = append(params, 1)
		case "nonaktif":
			conditions = append(conditions, "p.id_status = ?")
			params = append(params, 0)
		}
	}

	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY p.id_poli"

	rows, err := ps.DB.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var (
			idPoli, idStatus, jumlahTenkes int
			namaPoli, keterangan           string
			logoPoli                       sql.NullString
			createdAt                      time.Time
		)

		if err := rows.Scan(&idPoli, &idStatus, &namaPoli, &jumlahTenkes,
			&logoPoli, &keterangan, &createdAt); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}

		record := map[string]interface{}{
			"id_poli":       idPoli,
			"id_status":     idStatus,
			"nama_poli":     namaPoli,
			"jumlah_tenkes": jumlahTenkes,
			"logo_poli":     nil,
			"keterangan":    keterangan,
			"created_at":    createdAt.Format("02/01/2006"),
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

    // Proses logoPath:
    logoPath = strings.TrimSpace(logoPath)
    if logoPath == "" {
        // Jika tidak ada input, set ke "uploads/default.png"
        logoPath = "uploads/default.png"
    } else {
        // Pastikan prefiks uploads/ ada
        if !strings.HasPrefix(logoPath, "uploads/") {
            logoPath = "uploads/" + logoPath
        }
        // Tidak perlu cek nama kembar karena sudah unik dari controller
    }

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

	// Jika ada input logo (setelah trim), lakukan proses validasi dan update logo.
	if trimmedLogo := strings.TrimSpace(logoPoli); trimmedLogo != "" {
		// Ganti semua spasi dengan underscore.
		logoPoli = strings.ReplaceAll(trimmedLogo, " ", "_")

		// Cek apakah sudah ada logo dengan nama yang sama pada record lain (tidak menghitung record yang sedang diupdate).
		var count int
		queryCheck := "SELECT COUNT(*) FROM Poliklinik WHERE logo_poli = ? AND id_poli <> ?"
		if err = tx.QueryRow(queryCheck, logoPoli, idPoli).Scan(&count); err != nil {
			return fmt.Errorf("failed to check logo uniqueness: %v", err)
		}
		if count > 0 {
			return fmt.Errorf("Mohon ubah nama file anda! Nama file tidak boleh kembar!")
		}

		// Update Poliklinik termasuk kolom logo_poli.
		_, err = tx.Exec(`UPDATE Poliklinik 
		                  SET nama_poli = ?, keterangan = ?, logo_poli = ?, updated_at = NOW() 
		                  WHERE id_poli = ?`, namaPoli, keterangan, logoPoli, idPoli)
		if err != nil {
			return fmt.Errorf("failed to update Poliklinik: %v", err)
		}
	} else {
		// Jika tidak ada input logo, update hanya nama_poli dan keterangan.
		_, err = tx.Exec(`UPDATE Poliklinik 
		                  SET nama_poli = ?, keterangan = ?, updated_at = NOW() 
		                  WHERE id_poli = ?`, namaPoli, keterangan, idPoli)
		if err != nil {
			return fmt.Errorf("failed to update Poliklinik: %v", err)
		}
	}

	// Catat update di tabel Management_Poli (update kolom updated_by).
	_, err = tx.Exec(`UPDATE Management_Poli 
	                  SET updated_by = ? 
	                  WHERE id_poli = ?`, idManagement, idPoli)
	if err != nil {
		return fmt.Errorf("failed to update Management_Poli: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}


func (ps *PoliklinikService) GetActivePoliklinikList() ([]map[string]interface{}, error) {
	// Query hanya mengambil poliklinik yang aktif (id_status = 1) dan hanya kolom id_poli dan nama_poli.
	query := `
		SELECT id_poli, nama_poli
		FROM Poliklinik
		WHERE id_status = ?
		ORDER BY id_poli
	`
	rows, err := ps.DB.Query(query, 1)
	if err != nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var idPoli int
		var namaPoli string
		if err := rows.Scan(&idPoli, &namaPoli); err != nil {
			return nil, fmt.Errorf("scan error: %v", err)
		}
		record := map[string]interface{}{
			"id_poli":   idPoli,
			"nama_poli": namaPoli,
		}
		list = append(list, record)
	}
	return list, nil
}

// GetRuangListByPoliID retrieves the list of rooms for a specific poli from the database
func (ps *PoliklinikService) GetRuangListByPoliID(idPoli int) ([]map[string]interface{}, error) {
	query := `
			SELECT
					id_ruang,
					nama_ruang
			FROM Ruang
			WHERE id_poli = ?
	`
	rows, err := ps.DB.Query(query, idPoli)
	if err != nil {
			return nil, fmt.Errorf("query error: %v", err)
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
			var idRuang int
			var namaRuang string
			if err := rows.Scan(&idRuang, &namaRuang); err != nil {
					return nil, fmt.Errorf("scan error: %v", err)
			}
			record := map[string]interface{}{
					"id_ruang":   idRuang,
					"nama_ruang": namaRuang,
			}
			list = append(list, record)
	}
	if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("rows error: %v", err)
	}

	return list, nil
}