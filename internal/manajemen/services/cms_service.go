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

// ------------------------------------------------------------
// CreateCMSWithSections  (NO dummy subsection)
// ------------------------------------------------------------
func (svc *CMSService) CreateCMSWithSections(
	req models.CreateCMSRequest,
	mgmt models.ManagementCMS,
) (int64, error) {

	tx, err := svc.DB.Begin()
	if err != nil { return 0, err }
	defer func() { if err != nil { tx.Rollback() } }()

	now := time.Now()

	// 1) soft-delete CMS aktif di poli yang sama
	if _, err = tx.Exec(
		`UPDATE CMS SET deleted_at=? WHERE id_poli=? AND deleted_at IS NULL`,
		now, req.IDPoli); err != nil {
		return 0, err
	}

	// 2) header CMS
	res, err := tx.Exec(
		`INSERT INTO CMS (id_poli,title,created_at,updated_at) VALUES (?,?,?,?)`,
		req.IDPoli, req.Title, now, now)
	if err != nil { return 0, err }
	idCMS, _ := res.LastInsertId()

	// 3) validasi master Elements
	if err = svc.validateElementIDs(tx, req); err != nil { return 0, err }

// 4) loop section / subsection / element
for _, sec := range req.Sections {

	rSec, err2 := tx.Exec(
			`INSERT INTO CMS_Section (id_cms,title) VALUES (?,?)`,
			idCMS, sec.Title)
	if err2 != nil { return 0, err2 }
	idSection, _ := rSec.LastInsertId()

	// --- 4a. selalu simpan elemen langsung di section (jika ada) ---
	if len(sec.Elements) > 0 {
			if err = insertElements(tx, idSection, nil, sec.Elements); err != nil {
					return 0, err
			}
	}

	// --- 4b. kemudian proses setiap subseksi (jika ada) -------------
	for _, sub := range sec.Subsections {

			rSub, err3 := tx.Exec(
					`INSERT INTO CMS_Subsection (id_section,title) VALUES (?,?)`,
					idSection, sub.Title)
			if err3 != nil { return 0, err3 }
			idSub, _ := rSub.LastInsertId()

			if err = insertElements(tx, idSection, &idSub, sub.Elements); err != nil {
					return 0, err
			}
	}
}


	// 5) audit trail Management_CMS
	if _, err = tx.Exec(
		`INSERT INTO Management_CMS (id_management,id_cms,created_by,updated_by)
		 VALUES (?,?,?,?)`,
		mgmt.IDManagement, idCMS, mgmt.CreatedBy, mgmt.UpdatedBy); err != nil {
		return 0, err
	}

	return idCMS, tx.Commit()
}

// -------------------- helper private --------------------

// memvalidasi seluruh id_element yang dikirim user
func (svc *CMSService) validateElementIDs(tx *sql.Tx, req models.CreateCMSRequest) error {
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
	if len(elementIDs) == 0 {
		return nil
	}

	placeholders := make([]string, 0, len(elementIDs))
	args := make([]interface{}, 0, len(elementIDs))
	for id := range elementIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	query := fmt.Sprintf(
		`SELECT id_element FROM Elements WHERE id_element IN (%s)`,
		strings.Join(placeholders, ","))
	rows, err := tx.Query(query, args...)
	if err != nil { return err }
	defer rows.Close()

	existence := map[int]struct{}{}
	for rows.Next() {
		var id int
		rows.Scan(&id)
		existence[id] = struct{}{}
	}
	for id := range elementIDs {
		if _, ok := existence[id]; !ok {
			return errInvalidElementID{ID: id}
		}
	}
	return nil
}

func insertElements(
	tx *sql.Tx,
	idSection int64,
	idSub *int64,
	elems []models.ElementRequest,
) error {

	var subID interface{} = nil
	if idSub != nil { subID = *idSub }

	for _, el := range elems {

		var opts interface{}
		if len(el.ElementOptions) != 0 && string(el.ElementOptions) != "null" {
			opts = json.RawMessage(el.ElementOptions)
		}

		rE, err := tx.Exec(`
			INSERT INTO CMS_Elements
			  (id_section,id_subsection,element_label,element_name,
			   element_options,element_hint,is_required)
			VALUES (?,?,?,?,?,?,?)`,
			idSection, subID,
			el.ElementLabel, el.ElementName,
			opts, el.ElementHint, el.IsRequired)
		if err != nil { return err }

		idCE, _ := rE.LastInsertId()
		if _, err = tx.Exec(
			`INSERT INTO Detail_Element (id_element,id_cms_elements) VALUES (?,?)`,
			el.IDElement, idCE); err != nil {
			return err
		}
	}
	return nil
}


// ErrCMSNotFound indicates no CMS record exists for given ID
var ErrCMSNotFound = fmt.Errorf("cms not found")


var (
	ErrNoActiveCMSFound       = errors.New("no active CMS found for this polyclinic")
	ErrMultipleActiveCMSFound = errors.New("multiple active CMS found for this polyclinic")
)

func (svc *CMSService) GetActiveCMSIDByPoliID(poliID int) (int, error) {
	rows, err := svc.DB.Query(
		`SELECT id_cms FROM CMS WHERE id_poli = ? AND deleted_at IS NULL`, poliID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return 0, err
	}

	if len(ids) == 0 {
		return 0, ErrNoActiveCMSFound
	}
	if len(ids) > 1 {
		return 0, ErrMultipleActiveCMSFound
	}
	return ids[0], nil
}

func (svc *CMSService) GetActiveCMSDetailByPoliID(poliID int) (models.CMSDetailResponse, error) {
	idCMS, err := svc.GetActiveCMSIDByPoliID(poliID)
	if err != nil {
		return models.CMSDetailResponse{}, err
	}
	return svc.GetCMSDetailFull(idCMS)
}

// Existing GetCMSDetailFull method remains unchanged
func (svc *CMSService) GetCMSDetailFull(cmsID int) (models.CMSDetailResponse, error) {
	var resp models.CMSDetailResponse

	// 1) Header
	if err := svc.DB.QueryRow(
		`SELECT id_cms, title FROM CMS WHERE id_cms = ?`, cmsID,
	).Scan(&resp.IDCMS, &resp.Title); err != nil {
		if err == sql.ErrNoRows {
			return resp, ErrCMSNotFound
		}
		return resp, err
	}

	// 2) Detail elemen
	query := `
	  SELECT
	    e.id_cms_elements,
	    s.id_section,        s.title               AS section_title,
	    ss.id_subsection,    ss.title              AS subsection_title,
	    d.id_element,        el.type               AS element_type,
	    e.element_label,     e.element_name,
	    COALESCE(e.element_options,'') AS options,
	    COALESCE(e.element_hint,'')    AS hint,
	    e.is_required
	  FROM CMS_Section s
	    JOIN CMS_Elements   e  ON e.id_section   = s.id_section
	                            AND e.deleted_at IS NULL
	    LEFT JOIN CMS_Subsection ss ON ss.id_subsection = e.id_subsection
	                                  AND (ss.deleted_at IS NULL)
	    JOIN Detail_Element d  ON d.id_cms_elements = e.id_cms_elements
	    JOIN Elements       el ON el.id_element     = d.id_element
	  WHERE s.id_cms = ? AND s.deleted_at IS NULL
	  ORDER BY s.id_section, ss.id_subsection, e.id_cms_elements
	`
	rows, err := svc.DB.Query(query, cmsID)
	if err != nil {
		return resp, err
	}
	defer rows.Close()

	for rows.Next() {
		var det models.CMSElementDetail
		var (
			subID  sql.NullInt64
			subTit sql.NullString
			req    int
		)
		if err := rows.Scan(
			&det.IDCMSElement,
			&det.IDSection, &det.SectionTitle,
			&subID, &subTit,
			&det.IDElement, &det.ElementType,
			&det.Label, &det.Name,
			&det.Options, &det.Hint,
			&req,
		); err != nil {
			return resp, err
		}
		if subID.Valid {
			det.IDSubsection = int(subID.Int64)
		}
		det.SubTitle = subTit.String
		det.Required = req != 0
		resp.Elements = append(resp.Elements, det)
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


func (svc *CMSService) UpdateCMSWithSections(
	req models.UpdateCMSRequest,
	mgmt models.ManagementCMS,
) error {

	tx, err := svc.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	// 0) pastikan CMS eksis (aktif / soft-delete)
	var dummy int
	if err = tx.QueryRow(`SELECT id_cms FROM CMS WHERE id_cms = ?`, req.IDCMS).
		Scan(&dummy); err != nil {
		return fmt.Errorf("CMS tidak ditemukan")
	}

	// 1) update header
	if _, err = tx.Exec(`
		UPDATE CMS SET id_poli = ?, title = ?, updated_at = ?
		WHERE id_cms = ?`,
		req.IDPoli, req.Title, now, req.IDCMS); err != nil {
		return err
	}

	reName := regexp.MustCompile(`[^a-zA-Z0-9\s]`)

	// 2) proses setiap section
	for _, sec := range req.Sections {

		/* ---------- SECTION ---------- */
		if sec.Deleted && sec.IDSection != 0 {
			if _, err = tx.Exec(
				`UPDATE CMS_Section SET deleted_at = ? WHERE id_section = ?`,
				now, sec.IDSection); err != nil {
				return err
			}
			// ikut tandai child
			if _, err = tx.Exec(
				`UPDATE CMS_Subsection SET deleted_at = ? WHERE id_section = ?`,
				now, sec.IDSection); err != nil {
				return err
			}
			if _, err = tx.Exec(
				`UPDATE CMS_Elements SET deleted_at = ? WHERE id_section = ?`,
				now, sec.IDSection); err != nil {
				return err
			}
			continue
		}

		var idSection int64
		if sec.IDSection == 0 {
			res, err2 := tx.Exec(
				`INSERT INTO CMS_Section (id_cms, title) VALUES (?, ?)`,
				req.IDCMS, sec.Title)
			if err2 != nil {
				return err2
			}
			idSection, _ = res.LastInsertId()
		} else {
			idSection = int64(sec.IDSection)
			if _, err = tx.Exec(
				`UPDATE CMS_Section SET title = ? WHERE id_section = ?`,
				sec.Title, idSection); err != nil {
				return err
			}
		}

		// 2a. elemen top-level
		if len(sec.Elements) > 0 {
			if err = upsertElements(tx, idSection, nil,
				sec.Elements, reName, now); err != nil {
				return err
			}
		}

		// 2b. subseksi (jika ada)
		for _, sub := range sec.Subsections {

			if sub.Deleted && sub.IDSubsection != 0 {
				if _, err = tx.Exec(
					`UPDATE CMS_Subsection SET deleted_at = ? WHERE id_subsection = ?`,
					now, sub.IDSubsection); err != nil {
					return err
				}
				if _, err = tx.Exec(
					`UPDATE CMS_Elements SET deleted_at = ? WHERE id_subsection = ?`,
					now, sub.IDSubsection); err != nil {
					return err
				}
				continue
			}

			var idSub int64
			if sub.IDSubsection == 0 {
				res, err2 := tx.Exec(
					`INSERT INTO CMS_Subsection (id_section, title) VALUES (?, ?)`,
					idSection, sub.Title)
				if err2 != nil {
					return err2
				}
				idSub, _ = res.LastInsertId()
			} else {
				idSub = int64(sub.IDSubsection)
				if _, err = tx.Exec(
					`UPDATE CMS_Subsection SET title = ? WHERE id_subsection = ?`,
					sub.Title, idSub); err != nil {
					return err
				}
			}

			if err = upsertElements(tx, idSection, &idSub,
				sub.Elements, reName, now); err != nil {
				return err
			}
		}
	}

	// 3) audit trail
	if _, err = tx.Exec(`
		INSERT INTO Management_CMS (id_management, id_cms, created_by, updated_by)
		VALUES (?,?,?,?)
		ON DUPLICATE KEY UPDATE updated_by = ?`,
		mgmt.IDManagement, req.IDCMS, mgmt.CreatedBy, mgmt.UpdatedBy,
		mgmt.UpdatedBy); err != nil {
		return err
	}

	return tx.Commit()
}

/* ---------- helper ---------- */

// upsertElements   – create / update ringan / soft-delete
// jika id_element berubah, baris lama di-soft-delete & baris baru dibuat
func upsertElements(
	tx *sql.Tx,
	idSection int64,
	idSub *int64, // nil → NULL
	elements []models.ElementUpdate,
	reName *regexp.Regexp,
	now time.Time,
) error {

	for _, el := range elements {

		// eksplisit soft-delete
		if el.Deleted && el.IDCMSElements != 0 {
			if _, err := tx.Exec(
				`UPDATE CMS_Elements SET deleted_at = ? WHERE id_cms_elements = ?`,
				now, el.IDCMSElements); err != nil {
				return err
			}
			continue
		}

		// validasi master Elements
		var dummy int
		if err := tx.QueryRow(
			`SELECT id_element FROM Elements WHERE id_element = ?`,
			el.IDElement).Scan(&dummy); err != nil {
			return fmt.Errorf("invalid element ID: %d", el.IDElement)
		}

		// label → name
		clean := strings.TrimSpace(reName.ReplaceAllString(el.ElementLabel, ""))
		elemName := strings.ToLower(strings.ReplaceAll(clean, " ", "_"))

		// options
		var opts interface{}
		if len(el.ElementOptions) != 0 && string(el.ElementOptions) != "null" {
			opts = el.ElementOptions
		}

		// penentuan id_subsection (NULL utk top-level)
		var subID interface{} = nil
		if idSub != nil {
			subID = *idSub
		}

		/* ---------- apakah UPDATE ringan atau BREAKING? ---------- */
		breaking := false
		if el.IDCMSElements != 0 {
			var oldElementID int
			if err := tx.QueryRow(`
				SELECT d.id_element
				  FROM Detail_Element d
				  JOIN CMS_Elements e ON e.id_cms_elements = d.id_cms_elements
				 WHERE d.id_cms_elements = ?`, el.IDCMSElements).
				Scan(&oldElementID); err == nil {

				if oldElementID != el.IDElement {
					breaking = true
				}
			}
		}

		/* ---------- CREATE baru (first time atau breaking) ---------- */
		if el.IDCMSElements == 0 || breaking {

			// soft-delete baris lama jika breaking
			if breaking {
				if _, err := tx.Exec(
					`UPDATE CMS_Elements SET deleted_at = ? WHERE id_cms_elements = ?`,
					now, el.IDCMSElements); err != nil {
					return err
				}
			}

			rE, err := tx.Exec(`
				INSERT INTO CMS_Elements
				  (id_section, id_subsection, element_label, element_name,
				   element_options, element_hint, is_required)
				VALUES (?,?,?,?,?,?,?)`,
				idSection, subID, clean, elemName,
				opts, el.ElementHint, el.IsRequired)
			if err != nil {
				return err
			}
			newID, _ := rE.LastInsertId()

			if _, err := tx.Exec(
				`INSERT INTO Detail_Element (id_element, id_cms_elements)
				 VALUES (?, ?)`,
				el.IDElement, newID); err != nil {
				return err
			}
			continue
		}

		/* ---------- UPDATE ringan ---------- */
		if _, err := tx.Exec(`
			UPDATE CMS_Elements
			   SET element_label = ?, element_name = ?, element_options = ?,
			       element_hint  = ?, is_required   = ?, id_subsection   = ?
			 WHERE id_cms_elements = ?`,
			clean, elemName, opts, el.ElementHint, el.IsRequired,
			subID, el.IDCMSElements); err != nil {
			return err
		}

		if _, err := tx.Exec(
			`UPDATE Detail_Element
			    SET id_element = ?
			  WHERE id_cms_elements = ?`,
			el.IDElement, el.IDCMSElements); err != nil {
			return err
		}
	}
	return nil
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

var ErrAntrianNotFound = errors.New("antrian not found")

func (svc *CMSService) SaveAssessment(
	idAntrian, idPoli, idKaryawan int,
	input models.AssessmentInput,
) (idAssessment int64, err error) {

	tx, err := svc.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { if err != nil { _ = tx.Rollback() } }()

	/* --- 1. pastikan antrian ada & konsisten dgn poli --- */
	var idPasien, poliFromAntrian int
	err = tx.QueryRow(`SELECT id_pasien, id_poli FROM Antrian WHERE id_antrian = ?`,
		idAntrian).Scan(&idPasien, &poliFromAntrian)
	if err == sql.ErrNoRows {
		return 0, ErrAntrianNotFound
	}
	if err != nil {
		return 0, err
	}
	if poliFromAntrian != idPoli {
		return 0, fmt.Errorf("id_poli mismatch with antrian")
	}

	/* --- 2. cari CMS aktif utk poli --- */
	rows, err := tx.Query(`SELECT id_cms FROM CMS WHERE id_poli = ? AND deleted_at IS NULL`, idPoli)
	if err != nil {
		return 0, err
	}
	var activeCMS []int
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			return 0, err
		}
		activeCMS = append(activeCMS, id)
	}
	rows.Close()

	switch len(activeCMS) {
	case 0:
		var total int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM CMS WHERE id_poli = ?`, idPoli).Scan(&total); err != nil {
			return 0, err
		}
		if total == 0 {
			return 0, ErrCMSNeverCreated
		}
		return 0, ErrCMSNoneActive
	case 1:
		// ok
	default:
		return 0, ErrCMSMultipleActive
	}
	idCMS := activeCMS[0]

	/* --- 3. ambil elemen aktif & validasi jawaban --- */
	elemRows, err := tx.Query(`
		SELECT e.id_cms_elements, e.is_required
		FROM   CMS_Section s
		JOIN   CMS_Elements e ON e.id_section = s.id_section
		WHERE  s.id_cms = ? AND s.deleted_at IS NULL AND e.deleted_at IS NULL`,
		idCMS)
	if err != nil {
		return 0, err
	}
	required := map[int]struct{}{}
	allowed  := map[int]struct{}{}
	for elemRows.Next() {
		var idElem int
		var req int
		if err = elemRows.Scan(&idElem, &req); err != nil {
			return 0, err
		}
		allowed[idElem] = struct{}{}
		if req != 0 {
			required[idElem] = struct{}{}
		}
	}
	elemRows.Close()

	ansByID := map[int]models.CMSAnswer{}
	for _, a := range input.Answers {
		ansByID[a.IDCmsElement] = a
	}

	var unknown, missing []int
	for id := range ansByID {
		if _, ok := allowed[id]; !ok {
			unknown = append(unknown, id)
		}
	}
	for id := range required {
		if a, ok := ansByID[id]; !ok || isEmpty(a.Value) {
			missing = append(missing, id)
		}
	}
	if len(unknown) > 0 {
		return 0, fmt.Errorf("unknown id_cms_elements: %v", unknown)
	}
	if len(missing) > 0 {
		return 0, fmt.Errorf("required id_cms_elements empty: %v", missing)
	}

	/* --- 4. mapping khusus ruang_poli & diagnosis_awal_medis --- */
	var idRuang sql.NullInt64
	var idICD10 sql.NullString
	for _, a := range input.Answers {
		switch a.Name {
		case "ruang_poli":
			if v, ok := a.Value.(float64); ok {
				idRuang = sql.NullInt64{Int64: int64(v), Valid: true}
			}
		case "diagnosis_awal_medis":
			if v, ok := a.Value.(string); ok && v != "" {
				idICD10 = sql.NullString{String: v, Valid: true}
			}
		}
	}

	/* --- 5. serialisasi jawaban --- */
	raw, err := json.Marshal(input.Answers)
	if err != nil {
		return 0, err
	}

	/* --- 6. hapus assessment lama jika ada --- */
	_, err = tx.Exec(`DELETE FROM Assessment WHERE id_pasien = ? AND id_cms = ?`, idPasien, idCMS)
	if err != nil {
		return 0, err
	}

	/* --- 7. insert assessment baru --- */
	res, err := tx.Exec(`
		INSERT INTO Assessment
		  (id_pasien, id_karyawan, id_poli, id_ruang,
		   id_cms, id_icd10, hasil_assessment, created_at)
		VALUES (?,?,?,?,?,?,?,NOW())`,
		idPasien, idKaryawan, idPoli, idRuang,
		idCMS, idICD10, raw)
	if err != nil {
		return 0, err
	}
	idAssessment64, _ := res.LastInsertId()
	idAssessment = int64(idAssessment64)

/* ---------- 8. Update atau Insert Riwayat_Kunjungan ---------- */
    // Selalu update Riwayat_Kunjungan untuk id_antrian yang diberikan
    up, err := tx.Exec(`
        UPDATE Riwayat_Kunjungan
        SET id_assessment = ?
        WHERE id_antrian = ?`,
        idAssessment, idAntrian)
    if err != nil {
        return 0, err
    }

    affected, _ := up.RowsAffected()
    var idKunjungan int64

    if affected == 0 {
        // Tidak ada Riwayat_Kunjungan, buat baru
        var idRM string
        if err = tx.QueryRow(`
            SELECT rm.id_rm
            FROM Rekam_Medis rm
            JOIN Pasien p ON p.id_pasien = ?
            WHERE rm.id_pasien = p.id_pasien
            LIMIT 1`, idPasien).Scan(&idRM); err != nil {
            return 0, fmt.Errorf("pasien tidak memiliki rekam medis")
        }

        resRK, err2 := tx.Exec(`
            INSERT INTO Riwayat_Kunjungan
              (id_rm, id_antrian, id_assessment, created_at)
            VALUES (?, ?, ?, NOW())`,
            idRM, idAntrian, idAssessment)
        if err2 != nil {
            return 0, err2
        }
        idKunjungan, _ = resRK.LastInsertId()
    } else {
        // Ambil id_kunjungan yang baru saja diperbarui
        if err = tx.QueryRow(`
            SELECT id_kunjungan
            FROM Riwayat_Kunjungan
            WHERE id_antrian = ?`, idAntrian).Scan(&idKunjungan); err != nil {
            return 0, err
        }
    }

    /* ---------- 9. Update atau Insert Billing ---------- */
    // Cek apakah sudah ada billing untuk id_kunjungan
    var existingBillingID int
    err = tx.QueryRow(`
        SELECT id_billing
        FROM Billing
        WHERE id_kunjungan = ?`, idKunjungan).Scan(&existingBillingID)
    if err != nil && err != sql.ErrNoRows {
        return 0, err
    }

    if err == sql.ErrNoRows {
        // Tidak ada billing, lakukan INSERT
        _, err = tx.Exec(`
            INSERT INTO Billing
              (id_kunjungan, id_antrian, id_karyawan,
               id_assessment, id_status, created_at)
            VALUES (?, ?, ?, ?, 1, NOW())`,
            idKunjungan, idAntrian, idKaryawan, idAssessment)
        if err != nil {
            return 0, err
        }
    } else {
        // Billing sudah ada, lakukan UPDATE
        _, err = tx.Exec(`
            UPDATE Billing
            SET id_assessment = ?, id_karyawan = ?, updated_at = NOW()
            WHERE id_billing = ?`,
            idAssessment, idKaryawan, existingBillingID)
        if err != nil {
            return 0, err
        }
    }

    /* ---------- 10. Commit & return ---------- */
    if err = tx.Commit(); err != nil {
        return 0, err
    }
    return idAssessment, nil
}



/* ------- helper ------- */

func isEmpty(v interface{}) bool {
	switch vv := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(vv) == ""
	case float64, bool:
		return false
	default:
		return false
	}
}
var (
	ErrCMSMultipleActive = errors.New("more than one active CMS")
	ErrCMSNoneActive     = errors.New("no active CMS")
	ErrCMSNeverCreated   = errors.New("no CMS ever created for this poliklinik")
)

var ErrAssessmentAbsent  = errors.New("no assessment yet")


// GetRincianByAntrian mengambil keluhan_utama + mapping jawaban tertentu
func (svc *CMSService) GetRincianByAntrian(idAntrian int) (models.RincianAsesmen, error) {
    var r models.RincianAsesmen
    var idPasien int

    // 1. Ambil keluhan utama & id_pasien
    err := svc.DB.QueryRow(
        "SELECT keluhan_utama, id_pasien FROM Antrian WHERE id_antrian = ?", idAntrian,
    ).Scan(&r.KeluhanUtama, &idPasien)
    if err != nil {
        if err == sql.ErrNoRows { return r, ErrAntrianNotFound }
        return r, err
    }

    // 2. Ambil assessment terbaru pasien tsb
    var rawJSON []byte
    err = svc.DB.QueryRow(`
        SELECT hasil_assessment
        FROM Assessment
        WHERE id_pasien = ?
        ORDER BY created_at DESC
        LIMIT 1`, idPasien).Scan(&rawJSON)
    if err != nil {
        if err == sql.ErrNoRows { return r, ErrAssessmentAbsent }
        return r, err
    }

    // 3. Parse JSON array
    var answers []struct {
        Name  string      `json:"name"`
        Value interface{} `json:"value"`
    }
    if err := json.Unmarshal(rawJSON, &answers); err != nil {
        return r, err
    }

    // 4. Mapping field yang diminta
    for _, a := range answers {
        switch a.Name {
        case "riwayat_penyakit":
            r.RiwayatPenyakit = toString(a.Value)
        case "alergi":
            r.Alergi = toString(a.Value)
        case "jenis_reaksi":
            r.JenisReaksi = toString(a.Value)
        case "keadaan_umum_pasien":
            r.KeadaanUmumPasien = toString(a.Value)
        }
    }

    return r, nil
}

// helper convert interface{} -> string
func toString(v interface{}) string {
    switch t := v.(type) {
    case string:
        return t
    case float64:
        return fmt.Sprintf("%v", t)
    default:
        raw, _ := json.Marshal(t) // untuk array, dsb.
        return string(raw)
    }
}

func (svc *CMSService) GetAssessmentDetailFull(id int) (models.AssessmentDetailResponse, error) {
	var res models.AssessmentDetailResponse

	/* ---------------- 1. tarik header & blob ---------------- */
	var (
			cmsID int
			title string
			raw   string
	)
	err := svc.DB.QueryRow(`
			SELECT a.id_assessment,
						 a.id_cms,
						 c.title,
						 a.hasil_assessment
				FROM Assessment a
				JOIN CMS c ON c.id_cms = a.id_cms
			 WHERE a.id_assessment = ? AND a.deleted_at IS NULL`,
			id).Scan(&res.IDAssessment, &cmsID, &title, &raw)

	if err == sql.ErrNoRows {
			return res, ErrAssessmentNotFound
	}
	if err != nil {
			return res, err
	}
	res.IDCMS = cmsID
	res.Title = title

	/* ---------------- 2. decode jawaban --------------------- */
	var answers []struct {
			IDCMSElement int         `json:"id_cms_elements"`
			Label        string      `json:"label"`
			Name         string      `json:"name"`
			Value        interface{} `json:"value"`
	}
	if err = json.Unmarshal([]byte(raw), &answers); err != nil {
			return res, fmt.Errorf("bad stored JSON: %v", err)
	}
	if len(answers) == 0 {
			return res, nil // tidak ada jawaban
	}

	/* ---------------- 3. ambil meta-data semua id  ---------- */
	ids := make([]string, 0, len(answers))
	for _, a := range answers {
			ids = append(ids, fmt.Sprintf("%d", a.IDCMSElement))
	}
	metaQ := fmt.Sprintf(`
			SELECT e.id_cms_elements,
						 s.id_section,          s.title,
						 COALESCE(ss.id_subsection,0),
						 COALESCE(ss.title,''),  d.id_element,
						 el.type
				FROM CMS_Elements   e
				LEFT JOIN CMS_Section   s  ON s.id_section   = e.id_section
				LEFT JOIN CMS_Subsection ss ON ss.id_subsection = e.id_subsection
				LEFT JOIN Detail_Element d  ON d.id_cms_elements = e.id_cms_elements
				LEFT JOIN Elements       el ON el.id_element     = d.id_element
			 WHERE e.id_cms_elements IN (%s)`, strings.Join(ids, ","))
	rows, err := svc.DB.Query(metaQ)
	if err != nil {
			return res, err
	}
	defer rows.Close()

	meta := map[int]struct {
			secID   int
			secTtl  string
			subID   int
			subTtl  string
			elID    int
			elType  string
	}{}
	for rows.Next() {
			var (
					idElem int
					m      struct {
							secID   int
							secTtl  string
							subID   int
							subTtl  string
							elID    int
							elType  string
					}
			)
			if err := rows.Scan(&idElem,
					&m.secID, &m.secTtl, &m.subID, &m.subTtl, &m.elID, &m.elType); err != nil {
					return res, err
			}
			meta[idElem] = m
	}

	/* ---------------- 4. compose response ------------------- */
	for _, a := range answers {
			var det models.AssessmentElement

			det.IDCMSElement = a.IDCMSElement
			det.Label = a.Label
			det.Name  = a.Name
			det.Value = a.Value

			if m, ok := meta[a.IDCMSElement]; ok {
					det.IDSection    = m.secID
					det.SectionTitle = m.secTtl
					det.IDSubsection = m.subID
					det.SubTitle     = m.subTtl
					det.IDElement    = m.elID
					det.ElementType  = m.elType
			}
			res.Elements = append(res.Elements, det)
	}

	return res, nil
}
var ErrAssessmentNotFound = errors.New("assessment not found")


// MoveCMS memindahkan CMS yang nonaktif ke poli baru
func (svc *CMSService) MoveCMS(idCMS, newIDPoli int) error {
	tx, err := svc.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()

	// 1. Pastikan CMS ada dan nonaktif
	var deletedAt sql.NullTime
	var currentPoliID int
	err = tx.QueryRow(
		`SELECT id_poli, deleted_at FROM CMS WHERE id_cms = ?`,
		idCMS,
	).Scan(&currentPoliID, &deletedAt)
	if err == sql.ErrNoRows {
		return ErrCMSNotFound
	}
	if err != nil {
		return err
	}
	if !deletedAt.Valid {
		return ErrCMSNotInactive
	}

	// 2. Soft delete semua CMS aktif di poli target
	_, err = tx.Exec(
		`UPDATE CMS SET deleted_at = ?, updated_at = ? WHERE id_poli = ? AND deleted_at IS NULL`,
		now, now, newIDPoli,
	)
	if err != nil {
		return err
	}

	// 3. Perbarui id_poli pada CMS yang dipindahkan
	_, err = tx.Exec(
		`UPDATE CMS SET id_poli = ?, updated_at = ? WHERE id_cms = ?`,
		newIDPoli, now, idCMS,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

var (
	ErrCMSNotInactive  = errors.New("cms must be inactive to be moved")
	ErrCMSActiveInPoli = errors.New("another active cms exists in the target poli")
)