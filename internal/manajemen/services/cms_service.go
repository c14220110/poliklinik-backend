package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
)

type CMSService struct {
	DB *sql.DB
}

func NewCMSService(db *sql.DB) *CMSService {
	return &CMSService{DB: db}
}

// CreateCMSWithElements memasukkan data ke tabel CMS, CMS_Elements, dan Management_CMS.
// Jika sudah ada CMS untuk id_poli tersebut (dan deleted_at IS NULL), fungsi mengembalikan error.
func (cs *CMSService) CreateCMSWithElements(cms models.CMS, elements []models.CMSElement, managementInfo models.ManagementCMS) (int64, error) {
	// Cek apakah sudah ada CMS untuk id_poli tersebut
	var existingID int
	err := cs.DB.QueryRow("SELECT id_cms FROM CMS WHERE id_poli = ? AND deleted_at IS NULL LIMIT 1", cms.IDPoli).Scan(&existingID)
	if err == nil {
		// Jika tidak terjadi error, artinya ada record yang ditemukan.
		return 0, fmt.Errorf("CMS already exists for poliklinik with id_poli %d", cms.IDPoli)
	}
	if err != sql.ErrNoRows {
		// Jika error lain, kembalikan error
		return 0, fmt.Errorf("failed to check existing CMS: %v", err)
	}

	tx, err := cs.DB.Begin()
	if err != nil {
		return 0, err
	}

	// 1. Insert ke tabel CMS
	cmsInsert := `
		INSERT INTO CMS (id_poli, title, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`
	now := time.Now()
	res, err := tx.Exec(cmsInsert, cms.IDPoli, cms.Title, now, now)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to insert CMS: %v", err)
	}
	idCMS, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to get last insert id for CMS: %v", err)
	}

	// 2. Insert setiap elemen ke tabel CMS_Elements
	elemInsert := `
		INSERT INTO CMS_Elements (id_cms, element_type, element_label, element_name, element_options, is_required)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	for _, e := range elements {
		_, err := tx.Exec(elemInsert, idCMS, e.ElementType, e.ElementLabel, e.ElementName, e.ElementOptions, e.IsRequired)
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to insert CMS element: %v", err)
		}
	}

	// 3. Insert ke tabel Management_CMS
	managementInsert := `
		INSERT INTO Management_CMS (id_management, id_cms, created_by, updated_by)
		VALUES (?, ?, ?, ?)
	`
	_, err = tx.Exec(managementInsert, managementInfo.IDManagement, idCMS, managementInfo.CreatedBy, managementInfo.UpdatedBy)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to insert into Management_CMS: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return idCMS, nil
}


// GetCMSByPoliklinikID mengembalikan daftar CMS untuk suatu poliklinik.
func (cs *CMSService) GetCMSByPoliklinikID(poliID int) ([]models.CMSResponse, error) {
	query := `
		SELECT id_cms, id_poli, title, created_at
		FROM CMS
		WHERE id_poli = ?
		ORDER BY created_at DESC
	`
	rows, err := cs.DB.Query(query, poliID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var responses []models.CMSResponse
	for rows.Next() {
		var idCMS, idPoli int
		var title string
		var createdAt time.Time
		if err := rows.Scan(&idCMS, &idPoli, &title, &createdAt); err != nil {
			return nil, err
		}

		// Query elemen CMS untuk id_cms ini
		elemQuery := `
			SELECT id_elements, element_type, element_label, element_name, element_options, is_required
			FROM CMS_Elements
			WHERE id_cms = ?
		`
		elemRows, err := cs.DB.Query(elemQuery, idCMS)
		if err != nil {
			return nil, err
		}
		var elements []models.ElementInfo
		for elemRows.Next() {
			var e models.ElementInfo
			var isReq int
			if err := elemRows.Scan(&e.IDEelements, &e.ElementType, &e.ElementLabel, &e.ElementName, &e.ElementOptions, &isReq); err != nil {
				elemRows.Close()
				return nil, err
			}
			e.IsRequired = isReq != 0
			elements = append(elements, e)
		}
		elemRows.Close()

		// Query informasi management untuk CMS
		managementQuery := `
			SELECT id_management, created_by, updated_by
			FROM Management_CMS
			WHERE id_cms = ?
			LIMIT 1
		`
		var mInfo models.ManagementInfo
		err = cs.DB.QueryRow(managementQuery, idCMS).Scan(&mInfo.IDManagement, &mInfo.CreatedBy, &mInfo.UpdatedBy)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to query Management_CMS: %v", err)
		}

		response := models.CMSResponse{
			IDCMS:      idCMS,
			Title:      title,
			CreatedAt:  createdAt.Format(time.RFC3339),
			Management: mInfo,
			Elements:   elements,
		}
		responses = append(responses, response)
	}
	return responses, nil
}

// GetAllCMS mengembalikan daftar CMS yang dikelompokkan berdasarkan poliklinik.
func (cs *CMSService) GetAllCMS() ([]models.CMSGroup, error) {
	// Query semua poliklinik
	poliQuery := `
		SELECT id_poli, nama_poli
		FROM Poliklinik
	`
	rows, err := cs.DB.Query(poliQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []models.CMSGroup
	for rows.Next() {
		var group models.CMSGroup
		if err := rows.Scan(&group.IDPoli, &group.NamaPoli); err != nil {
			return nil, err
		}
		// Ambil CMS untuk tiap poliklinik
		cmsList, err := cs.GetCMSByPoliklinikID(group.IDPoli)
		if err != nil {
			return nil, err
		}
		group.CMS = cmsList
		groups = append(groups, group)
	}
	return groups, nil
}


// UpdateCMSWithElements mengupdate record CMS, CMS_Elements, dan Management_CMS.
// Pendekatan: update CMS (title dan updated_at), hapus elemen lama, insert elemen baru,
// dan update kolom updated_by di Management_CMS berdasarkan token JWT saat ini.
func (cs *CMSService) UpdateCMSWithElements(idCMS int, newTitle string, elements []models.CMSElement, managementInfo models.ManagementCMS) error {
	tx, err := cs.DB.Begin()
	if err != nil {
		return err
	}

	// 1. Update record CMS (title dan updated_at)
	updateCMSQuery := `
		UPDATE CMS
		SET title = ?, updated_at = ?
		WHERE id_cms = ?
	`
	now := time.Now()
	_, err = tx.Exec(updateCMSQuery, newTitle, now, idCMS)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update CMS: %v", err)
	}

	// 2. Hapus elemen lama di CMS_Elements
	deleteElementsQuery := `DELETE FROM CMS_Elements WHERE id_cms = ?`
	_, err = tx.Exec(deleteElementsQuery, idCMS)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete old CMS elements: %v", err)
	}

	// 3. Insert elemen baru ke CMS_Elements
	elemInsert := `
		INSERT INTO CMS_Elements (id_cms, element_type, element_label, element_name, element_options, is_required)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	for _, e := range elements {
		_, err := tx.Exec(elemInsert, idCMS, e.ElementType, e.ElementLabel, e.ElementName, e.ElementOptions, e.IsRequired)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert CMS element: %v", err)
		}
	}

	// 4. Update Management_CMS: hanya update kolom updated_by berdasarkan token JWT
	updateManagementQuery := `
		UPDATE Management_CMS
		SET updated_by = ?
		WHERE id_cms = ?
	`
	_, err = tx.Exec(updateManagementQuery, managementInfo.UpdatedBy, idCMS)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update Management_CMS: %v", err)
	}

	return tx.Commit()
}
