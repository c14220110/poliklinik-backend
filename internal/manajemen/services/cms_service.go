package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
)

type CMSService struct {
	DB *sql.DB
}

func NewCMSService(db *sql.DB) *CMSService {
	return &CMSService{DB: db}
}

var (
	// ErrCMSAlreadyExists returned when a CMS for the given poliklinik already exists
	ErrCMSAlreadyExists = errors.New("cms already exists for this poliklinik")
	// ErrInvalidElementID returned when request contains an element ID not in master Elements table
	ErrInvalidElementID = errors.New("invalid element ID in request")
)

// CreateCMSWithSections membuat CMS beserta section, subsection, elements, detail_element, dan management
func (svc *CMSService) CreateCMSWithSections(
	req models.CreateCMSRequest,
	mgmt models.ManagementCMS,
) (int64, error) {
	// 1. cek duplikat CMS untuk id_poli
	var exists int
	err := svc.DB.QueryRow(
		"SELECT id_cms FROM CMS WHERE id_poli = ? AND deleted_at IS NULL",
		req.IDPoli,
	).Scan(&exists)
	if err == nil {
		return 0, ErrCMSAlreadyExists
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	// 2. kumpulkan semua id_element dari request
	elementIDs := map[int]struct{}{}
	for _, sec := range req.Sections {
		for _, el := range sec.Elements {
			elementIDs[el.IDElement] = struct{}{}
		}
		for _, sub := range sec.Subsections {
			for _, el := range sub.Elements {
				elementIDs[el.IDElement] = struct{}{}
			}
		}
	}
	// jika ada id_element, verifikasi keberadaannya
	if len(elementIDs) > 0 {
		placeholders := make([]string, 0, len(elementIDs))
		args := make([]interface{}, 0, len(elementIDs))
		for id := range elementIDs {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}
		// bangun IN clause
		inClause := strings.Join(placeholders, ",")
		query := fmt.Sprintf("SELECT id_element FROM Elements WHERE id_element IN (%s)", inClause)
		rows, err := svc.DB.Query(query, args...)
		if err != nil {
			return 0, err
		}
		defer rows.Close()

		existing := map[int]struct{}{}
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				return 0, err
			}
			existing[id] = struct{}{}
		}
		// cek ada yang invalid
		for id := range elementIDs {
			if _, ok := existing[id]; !ok {
				return 0, fmt.Errorf("%w: %d", ErrInvalidElementID, id)
			}
		}
	}

	// 3. mulai transaksi
	tx, err := svc.DB.Begin()
	if err != nil {
		return 0, err
	}

	// insert CMS
	now := time.Now()
	res, err := tx.Exec(
		`INSERT INTO CMS (id_poli, title, created_at, updated_at)
		 VALUES (?, ?, ?, ?)`,
		req.IDPoli, req.Title, now, now,
	)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	idCMS, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// insert sections, subsections, elements, detail_element
	for _, sec := range req.Sections {
		// insert section
		rSec, err := tx.Exec(
			`INSERT INTO CMS_Section (id_cms, title) VALUES (?, ?)`,
			idCMS, sec.Title,
		)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		idSection, err := rSec.LastInsertId()
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		// siapkan subsections (jika none, treat elements at section level)
		subsecs := sec.Subsections
		if len(subsecs) == 0 && len(sec.Elements) > 0 {
			subsecs = []models.SubsectionRequest{{Title: "", Elements: sec.Elements}}
		}

		for _, sub := range subsecs {
			// insert subsection
			rSub, err := tx.Exec(
				`INSERT INTO CMS_Subsection (id_section, title) VALUES (?, ?)`,
				idSection, sub.Title,
			)
			if err != nil {
				tx.Rollback()
				return 0, err
			}
			idSub, err := rSub.LastInsertId()
			if err != nil {
				tx.Rollback()
				return 0, err
			}

			// insert elements
			for _, el := range sub.Elements {
				// element_options JSON raw (string atau null)
				var opts interface{}
				if string(el.ElementOptions) == "null" || len(el.ElementOptions) == 0 {
					opts = nil
				} else {
					opts = json.RawMessage(el.ElementOptions)
				}

				// insert into CMS_Elements
				rEl, err := tx.Exec(
					`INSERT INTO CMS_Elements
					 (id_section, id_subsection, element_label, element_name, element_options, element_hint, is_required)
					 VALUES (?, ?, ?, ?, ?, ?, ?)`,
					idSection, idSub,
					el.ElementLabel, el.ElementName,
					opts, el.ElementHint, el.IsRequired,
				)
				if err != nil {
					tx.Rollback()
					return 0, err
				}
				idCMSEl, err := rEl.LastInsertId()
				if err != nil {
					tx.Rollback()
					return 0, err
				}

				// insert detail_element
				_, err = tx.Exec(
					`INSERT INTO Detail_Element (id_element, id_cms_elements) VALUES (?, ?)`,
					el.IDElement, idCMSEl,
				)
				if err != nil {
					tx.Rollback()
					return 0, err
				}
			}
		}
	}

	// insert management
	_, err = tx.Exec(
		`INSERT INTO Management_CMS (id_management, id_cms, created_by, updated_by)
		 VALUES (?, ?, ?, ?)`,
		mgmt.IDManagement, idCMS, mgmt.CreatedBy, mgmt.UpdatedBy,
	)
	if err != nil {
		tx.Rollback()
		return 0, err
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


func (cs *CMSService) GetAllCMS() ([]models.CMSFlat, error) {
    query := `
        SELECT p.id_poli, p.nama_poli, c.id_cms
        FROM Poliklinik p
        LEFT JOIN CMS c ON p.id_poli = c.id_poli
    `
    rows, err := cs.DB.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var cmsFlatList []models.CMSFlat
    for rows.Next() {
        var cmsFlat models.CMSFlat
        var idCms sql.NullInt64 // Menangani nilai NULL dari database
        err := rows.Scan(&cmsFlat.IDPoli, &cmsFlat.NamaPoli, &idCms)
        if err != nil {
            return nil, err
        }
        if idCms.Valid {
            id := int(idCms.Int64)
            cmsFlat.IDCms = &id
        } else {
            cmsFlat.IDCms = nil
        }
        cmsFlatList = append(cmsFlatList, cmsFlat)
    }
    return cmsFlatList, nil
}


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
        INSERT INTO CMS_Elements (id_cms, section_name, element_type, element_label, element_name, element_options, is_required)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `
    for _, e := range elements {
        _, err := tx.Exec(elemInsert, idCMS, e.SectionName, e.ElementType, e.ElementLabel, e.ElementName, e.ElementOptions, e.IsRequired)
        if err != nil {
            tx.Rollback()
            return fmt.Errorf("failed to insert CMS element: %v", err)
        }
    }

    // 4. Update Management_CMS: set updated_by dengan id_management (integer) dari managementInfo
    updateManagementQuery := `
        UPDATE Management_CMS
        SET updated_by = ?
        WHERE id_cms = ?
    `
    _, err = tx.Exec(updateManagementQuery, managementInfo.IDManagement, idCMS)
    if err != nil {
        tx.Rollback()
        return fmt.Errorf("failed to update Management_CMS: %v", err)
    }

    return tx.Commit()
}