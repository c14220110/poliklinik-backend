package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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

// ErrInvalidElementID returned when request contains an element ID not in master Elements
type errInvalidElementID struct{ ID int }
func (e errInvalidElementID) Error() string { return fmt.Sprintf("invalid element ID: %d", e.ID) }

// CreateCMSWithSections inserts a new CMS, soft-deletes any active CMS for the same poli,
// and populates sections, subsections, elements, Detail_Element, and Management_CMS.
func (svc *CMSService) CreateCMSWithSections(
	req models.CreateCMSRequest,
	mgmt models.ManagementCMS,
) (int64, error) {
	// begin transaction
	tx, err := svc.DB.Begin()
	if err != nil {
		return 0, err
	}
	now := time.Now()

	// 1) Soft-delete existing CMS for this poli
	if _, err := tx.Exec(
		`UPDATE CMS SET deleted_at = ? WHERE id_poli = ? AND deleted_at IS NULL`,
		now, req.IDPoli,
	); err != nil {
		tx.Rollback()
		return 0, err
	}

	// 2) Insert new CMS record
	res, err := tx.Exec(
		`INSERT INTO CMS (id_poli, title, created_at, updated_at) VALUES (?, ?, ?, ?)`,
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

	// 3) Validate element IDs against master Elements table
	elementIDs := make(map[int]struct{})
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
	if len(elementIDs) > 0 {
		placeholders := make([]string, 0, len(elementIDs))
		args := make([]interface{}, 0, len(elementIDs))
		for id := range elementIDs {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}
		query := fmt.Sprintf("SELECT id_element FROM Elements WHERE id_element IN (%s)", strings.Join(placeholders, ","))
		exRows, err := svc.DB.Query(query, args...)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		defer exRows.Close()

		existing := make(map[int]struct{})
		for exRows.Next() {
			var id int
			exRows.Scan(&id)
			existing[id] = struct{}{}
		}
		for id := range elementIDs {
			if _, ok := existing[id]; !ok {
				tx.Rollback()
				return 0, errInvalidElementID{ID: id}
			}
		}
	}

	// 4) Insert sections, subsections, elements, and Detail_Element
	for _, sec := range req.Sections {
		// section
		rS, err := tx.Exec(`INSERT INTO CMS_Section (id_cms, title) VALUES (?, ?)`, idCMS, sec.Title)
		if err != nil { tx.Rollback(); return 0, err }
		idSection, _ := rS.LastInsertId()

		subs := sec.Subsections
		if len(subs) == 0 && len(sec.Elements) > 0 {
			subs = []models.SubsectionRequest{{Title: "", Elements: sec.Elements}}
		}
		for _, sub := range subs {
			rSub, err := tx.Exec(`INSERT INTO CMS_Subsection (id_section, title) VALUES (?, ?)`, idSection, sub.Title)
			if err != nil { tx.Rollback(); return 0, err }
			idSub, _ := rSub.LastInsertId()

			for _, el := range sub.Elements {
				// prepare element_options
				var opts interface{}
				if len(el.ElementOptions) == 0 || string(el.ElementOptions) == "null" {
					opts = nil
				} else {
					opts = json.RawMessage(el.ElementOptions)
				}

				// insert CMS_Elements
				rE, err := tx.Exec(
					`INSERT INTO CMS_Elements
					 (id_section, id_subsection, element_label, element_name, element_options, element_hint, is_required)
					 VALUES (?, ?, ?, ?, ?, ?, ?)`,
					idSection, idSub,
					el.ElementLabel, el.ElementName,
					opts, el.ElementHint, el.IsRequired,
				)
				if err != nil { tx.Rollback(); return 0, err }
				idCE, _ := rE.LastInsertId()

				// link to master element
				if _, err := tx.Exec(`INSERT INTO Detail_Element (id_element, id_cms_elements) VALUES (?, ?)`, el.IDElement, idCE); err != nil {
					tx.Rollback(); return 0, err
				}
			}
		}
	}

	// 5) Insert Management_CMS
	if _, err := tx.Exec(
		`INSERT INTO Management_CMS (id_management, id_cms, created_by, updated_by) VALUES (?, ?, ?, ?)`,
		mgmt.IDManagement, idCMS, mgmt.CreatedBy, mgmt.UpdatedBy,
	); err != nil {
		tx.Rollback()
		return 0, err
	}

	// commit
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return idCMS, nil
}


// ErrCMSNotFound indicates no CMS record exists for given ID
var ErrCMSNotFound = fmt.Errorf("cms not found")



// GetCMSDetailByID returns a detailed CMS including all elements for given cmsID
func (svc *CMSService) GetCMSDetailByID(cmsID int) (models.CMSDetailResponse, error) {
	var resp models.CMSDetailResponse
	// 1) Fetch CMS header
	headerQ := `SELECT id_cms, title FROM CMS WHERE id_cms = ?`
	if err := svc.DB.QueryRow(headerQ, cmsID).Scan(&resp.IDCMS, &resp.Title); err != nil {
		if err == sql.ErrNoRows {
			return resp, ErrCMSNotFound
		}
		return resp, err
	}

	// 2) Fetch all elements via join through CMS_Section
	eleQ := `
		SELECT
		e.id_cms_elements,
		e.id_section,
		e.id_subsection,
		e.element_label,
		e.element_name,
		e.element_options,
		e.element_hint,
		e.is_required
		FROM CMS_Elements e
		JOIN CMS_Section s ON e.id_section = s.id_section
		WHERE s.id_cms = ?
		ORDER BY e.id_cms_elements
	`
	rows, err := svc.DB.Query(eleQ, cmsID)
	if err != nil {
		return resp, err
	}
	defer rows.Close()

	for rows.Next() {
		var e models.CMSElementDetail
		var reqInt int
		var optNull sql.NullString
		var hintNull sql.NullString
		if err := rows.Scan(
			&e.IDCMSElement,
			&e.IDSection,
			&e.IDSubsection,
			&e.Label,
			&e.Name,
			&optNull,
			&hintNull,
			&reqInt,
		); err != nil {
			return resp, err
		}
		e.Required = reqInt != 0
		if optNull.Valid {
			e.Options = optNull.String
		}
		if hintNull.Valid {
			e.Hint = hintNull.String
		}
		resp.Elements = append(resp.Elements, e)
	}
	return resp, nil
}

// ErrNoCMSForPoli is returned when no CMS records exist under the given poliklinik ID
var ErrNoCMSForPoli = fmt.Errorf("no CMS entries found for this poliklinik")
// GetCMSListByPoli returns a list of CMS entries with status (active/non-active) for a given poliklinik
func (svc *CMSService) GetCMSListByPoli(poliID int) ([]models.CMSListItem, error) {
	query := `
		SELECT id_cms, title, deleted_at
		FROM CMS
		WHERE id_poli = ?
		ORDER BY created_at DESC
	`
	rows, err := svc.DB.Query(query, poliID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.CMSListItem
	for rows.Next() {
		var (
			idCMS     int
			title     string
			deletedAt sql.NullTime
		)
		if err := rows.Scan(&idCMS, &title, &deletedAt); err != nil {
			return nil, err
		}
		status := "aktif"
		if deletedAt.Valid {
			status = "nonaktif"
		}
		list = append(list, models.CMSListItem{
			IDCMS:  idCMS,
			Title:  title,
			Status: status,
		})
	}
	if len(list) == 0 {
		return nil, ErrNoCMSForPoli
	}
	return list, nil
}


// services/cms_service.go
// UpdateCMSWithSections memperbarui CMS beserta seluruh hirarki­nya.
func (svc *CMSService) UpdateCMSWithSections(
	req models.UpdateCMSRequest,
	mgmt models.ManagementCMS,
) error {

	tx, err := svc.DB.Begin()
	if err != nil { return err }
	now := time.Now()

	// 0) Pastikan CMS ada         ⬅️  HAPUS filter deleted_at
var dummy int
if err := tx.QueryRow(`SELECT id_cms FROM CMS WHERE id_cms = ?`,   // ← cukup ini
	req.IDCMS).Scan(&dummy); err != nil {
	tx.Rollback()
	return fmt.Errorf("CMS tidak ditemukan") // masih valid utk id yg memang tidak ada
}


	// 1) Update header CMS
	if _, err := tx.Exec(`
		UPDATE CMS SET id_poli = ?, title = ?, updated_at = ?
		WHERE id_cms = ?`,
		req.IDPoli, req.Title, now, req.IDCMS); err != nil {
		tx.Rollback(); return err
	}

	// 2) Bangun set ID yg datang dari frontend ─ dipakai utk deteksi delete implisit
	payloadSecIDs   := map[int]struct{}{}
	payloadSubIDs   := map[int]struct{}{}
	payloadElemIDs  := map[int]struct{}{}

	// 3) Validasi ID master Elements sekaligus upsert hirarki
	reName := regexp.MustCompile(`[^a-zA-Z0-9\s]`) // utk generate name

	for _, sec := range req.Sections {
		if sec.Deleted {
			// soft-delete section → cascade ke children via FK ON DELETE CASCADE
			if _, err := tx.Exec(`UPDATE CMS_Section SET deleted_at=? WHERE id_section=?`,
				now, sec.IDSection); err != nil { tx.Rollback(); return err }
			continue
		}
		var idSection int64
		if sec.IDSection == 0 {
			res, err := tx.Exec(`INSERT INTO CMS_Section (id_cms, title) VALUES (?,?)`,
				req.IDCMS, sec.Title)
			if err != nil { tx.Rollback(); return err }
			idSection, _ = res.LastInsertId()
		} else {
			payloadSecIDs[sec.IDSection] = struct{}{}
			idSection = int64(sec.IDSection)
			if _, err := tx.Exec(`UPDATE CMS_Section SET title=? WHERE id_section=?`,
				sec.Title, idSection); err != nil { tx.Rollback(); return err }
		}

		// -------- Subsection loop --------
		subs := sec.Subsections
		if len(subs) == 0 && len(sec.Elements) > 0 {
			subs = []models.SubsectionUpdate{{ /*dummy*/ Elements: sec.Elements }}
		}
		for _, sub := range subs {
			if sub.Deleted && sub.IDSubsection != 0 {
				if _, err := tx.Exec(`UPDATE CMS_Subsection SET deleted_at=? WHERE id_subsection=?`,
					now, sub.IDSubsection); err != nil { tx.Rollback(); return err }
				continue
			}

			var idSub int64
			if sub.IDSubsection == 0 {
				res, err := tx.Exec(`INSERT INTO CMS_Subsection (id_section, title) VALUES (?,?)`,
					idSection, sub.Title)
				if err != nil { tx.Rollback(); return err }
				idSub, _ = res.LastInsertId()
			} else {
				payloadSubIDs[sub.IDSubsection] = struct{}{}
				idSub = int64(sub.IDSubsection)
				if _, err := tx.Exec(`UPDATE CMS_Subsection SET title=? WHERE id_subsection=?`,
					sub.Title, idSub); err != nil { tx.Rollback(); return err }
			}

			// -------- Element loop --------
			for _, el := range sub.Elements {
				if el.Deleted && el.IDCMSElements != 0 {
					if _, err := tx.Exec(`DELETE FROM CMS_Elements WHERE id_cms_elements=?`,
						el.IDCMSElements); err != nil { tx.Rollback(); return err }
					continue
				}
				// --- validasi id_element ada di master Elements
				var tmp int
				if err := tx.QueryRow(`SELECT id_element FROM Elements WHERE id_element=?`,
					el.IDElement).Scan(&tmp); err != nil {
					tx.Rollback(); return fmt.Errorf("invalid element ID: %d", el.IDElement)
				}

				// --- siapkan element_name
				clean := strings.TrimSpace(reName.ReplaceAllString(el.ElementLabel, ""))
				elemName := strings.ToLower(strings.ReplaceAll(clean, " ", "_"))

				opts := interface{}(nil)
				if len(el.ElementOptions) != 0 && string(el.ElementOptions) != "null" {
					opts = el.ElementOptions
				}

				if el.IDCMSElements == 0 { // -------- CREATE
					res, err := tx.Exec(`
						INSERT INTO CMS_Elements
						  (id_section,id_subsection,element_label,element_name,
						   element_options,element_hint,is_required)
						VALUES (?,?,?,?,?,?,?)`,
						idSection, idSub, clean, elemName, opts, el.ElementHint, el.IsRequired)
					if err != nil { tx.Rollback(); return err }
					newIDElem, _ := res.LastInsertId()
					if _, err := tx.Exec(
						`INSERT INTO Detail_Element (id_element,id_cms_elements) VALUES (?,?)`,
						el.IDElement, newIDElem); err != nil { tx.Rollback(); return err }
				} else { // ---------------------- UPDATE
					payloadElemIDs[el.IDCMSElements] = struct{}{}
					if _, err := tx.Exec(`
						UPDATE CMS_Elements
						   SET element_label=?, element_name=?, element_options=?,
						       element_hint=?, is_required=?
						 WHERE id_cms_elements=?`,
						clean, elemName, opts, el.ElementHint, el.IsRequired,
						el.IDCMSElements); err != nil { tx.Rollback(); return err }

					// perbarui Detail_Element jika id_element berubah
					if _, err := tx.Exec(`
						UPDATE Detail_Element SET id_element=? WHERE id_cms_elements=?`,
						el.IDElement, el.IDCMSElements); err != nil { tx.Rollback(); return err }
				}
			}
		}
	}

	// 4) Update Management_CMS (audit trail)
	if _, err := tx.Exec(`
		INSERT INTO Management_CMS (id_management,id_cms,created_by,updated_by)
		VALUES (?,?,?,?)
		ON DUPLICATE KEY UPDATE updated_by=?`,
		mgmt.IDManagement, req.IDCMS, mgmt.CreatedBy, mgmt.UpdatedBy,
		mgmt.UpdatedBy); err != nil { tx.Rollback(); return err }

	return tx.Commit()
}


// Custom errors
var (
    ErrCMSAlreadyActive   = errors.New("cms already active")
    ErrOtherCMSActive     = errors.New("another cms is already active for this poliklinik")
    ErrCMSAlreadyInactive = errors.New("cms already inactive")
		ErrCMSNotActive     = errors.New("cms already non-active")
)

// ActivateCMS sets deleted_at=NULL for given cmsID if no other active CMS exists in the same poli
func (svc *CMSService) ActivateCMS(cmsID int) error {
    tx, err := svc.DB.Begin()
    if err != nil {
        return err
    }

    var idPoli sql.NullInt64
    var deletedAt sql.NullTime
    // fetch cms and state
    row := tx.QueryRow("SELECT id_poli, deleted_at FROM CMS WHERE id_cms = ?", cmsID)
    if err = row.Scan(&idPoli, &deletedAt); err != nil {
        tx.Rollback()
        if err == sql.ErrNoRows {
            return ErrCMSNotFound
        }
        return err
    }

    // already active?
    if !deletedAt.Valid {
        tx.Rollback()
        return ErrCMSAlreadyActive
    }

    // ensure no other active CMS in same poli
    if idPoli.Valid {
        var other int
        err = tx.QueryRow("SELECT id_cms FROM CMS WHERE id_poli = ? AND deleted_at IS NULL LIMIT 1", idPoli.Int64).Scan(&other)
        if err == nil {
            tx.Rollback()
            return ErrOtherCMSActive
        } else if err != sql.ErrNoRows {
            tx.Rollback()
            return err
        }
    }

    // activate this cms
    _, err = tx.Exec("UPDATE CMS SET deleted_at = NULL, updated_at=? WHERE id_cms = ?", time.Now(), cmsID)
    if err != nil {
        tx.Rollback()
        return err
    }

    return tx.Commit()
}

// DeactivateCMS sets deleted_at=NOW() for given cmsID
func (svc *CMSService) DeactivateCMS(cmsID int) error {
    res, err := svc.DB.Exec("UPDATE CMS SET deleted_at = ? , updated_at = ? WHERE id_cms = ? AND deleted_at IS NULL", time.Now(), time.Now(), cmsID)
    if err!=nil { return err }
    rows, _ := res.RowsAffected()
    if rows==0 { // already non active or not found
        var exists int
        if err := svc.DB.QueryRow("SELECT 1 FROM CMS WHERE id_cms = ?", cmsID).Scan(&exists); err==sql.ErrNoRows {
            return ErrCMSNotFound
        }
        return ErrCMSNotActive
    }
    return nil
}
