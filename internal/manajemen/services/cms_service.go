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

// CreateCMSWithSections membuat record CMS baru, men‑soft‑delete CMS aktif di poli yg sama,
// lalu menyimpan section, OPTIONAL subsection, elements (tanpa dummy), Detail_Element, dan Management_CMS.
func (svc *CMSService) CreateCMSWithSections(
	req models.CreateCMSRequest,
	mgmt models.ManagementCMS,
) (int64, error) {

	tx, err := svc.DB.Begin()
	if err != nil {
		return 0, err
	}
	now := time.Now()

	// 1) Soft‑delete CMS lama di poli yg sama
	if _, err := tx.Exec(
		`UPDATE CMS SET deleted_at = ? WHERE id_poli = ? AND deleted_at IS NULL`,
		now, req.IDPoli); err != nil {
		tx.Rollback()
		return 0, err
	}

	// 2) Insert header CMS baru
	res, err := tx.Exec(
		`INSERT INTO CMS (id_poli, title, created_at, updated_at) VALUES (?,?,?,?)`,
		req.IDPoli, req.Title, now, now)
	if err != nil { tx.Rollback(); return 0, err }
	idCMS, _ := res.LastInsertId()

	// 3) Validasi id_element
	if err := svc.validateElementIDs(tx, req); err != nil {
		tx.Rollback(); return 0, err
	}

	// 4) Insert Section + (optional) Subsection + Elements
	for _, sec := range req.Sections {

		rSec, err := tx.Exec(`INSERT INTO CMS_Section (id_cms, title) VALUES (?,?)`,
			idCMS, sec.Title)
		if err != nil { tx.Rollback(); return 0, err }
		idSection, _ := rSec.LastInsertId()

		if len(sec.Subsections) > 0 {
			// ada subseksi normal
			for _, sub := range sec.Subsections {
				rSub, err := tx.Exec(`INSERT INTO CMS_Subsection (id_section, title) VALUES (?,?)`,
					idSection, sub.Title)
				if err != nil { tx.Rollback(); return 0, err }
				idSub, _ := rSub.LastInsertId()

				if err := insertElements(tx, idSection, &idSub, sub.Elements); err != nil {
					tx.Rollback(); return 0, err
				}
			}
		} else if len(sec.Elements) > 0 {
			// elemen langsung di level section (id_subsection = NULL)
			if err := insertElements(tx, idSection, nil, sec.Elements); err != nil {
				tx.Rollback(); return 0, err
			}
		}
	}

	// 5) Audit trail ke Management_CMS
	if _, err := tx.Exec(`
		INSERT INTO Management_CMS (id_management,id_cms,created_by,updated_by)
		VALUES (?,?,?,?)`,
		mgmt.IDManagement, idCMS, mgmt.CreatedBy, mgmt.UpdatedBy); err != nil {
		tx.Rollback(); return 0, err
	}

	// commit
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return idCMS, nil
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

// insertElements menyisipkan array elemen ke CMS_Elements + Detail_Element.
// idSub=nil ➜ kolom id_subsection disimpan NULL.
func insertElements(
	tx *sql.Tx,
	idSection int64,
	idSub *int64,
	elements []models.ElementRequest,
) error {

	for _, el := range elements {

		// siapkan element_options (nullable)
		var opts interface{}
		if len(el.ElementOptions) != 0 && string(el.ElementOptions) != "null" {
			opts = json.RawMessage(el.ElementOptions)
		}

		// subID = NULL jika tidak ada subsection
		var subID interface{} = nil
		if idSub != nil { subID = *idSub }

		rE, err := tx.Exec(`
			INSERT INTO CMS_Elements
			  (id_section, id_subsection, element_label, element_name,
			   element_options, element_hint, is_required)
			VALUES (?,?,?,?,?,?,?)`,
			idSection, subID,
			el.ElementLabel, el.ElementName,
			opts, el.ElementHint, el.IsRequired)
		if err != nil { return err }

		idCE, _ := rE.LastInsertId()
		if _, err := tx.Exec(`
			INSERT INTO Detail_Element (id_element, id_cms_elements) VALUES (?,?)`,
			el.IDElement, idCE); err != nil { return err }
	}
	return nil
}


// ErrCMSNotFound indicates no CMS record exists for given ID
var ErrCMSNotFound = fmt.Errorf("cms not found")



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
	if err != nil { return resp, err }
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
		if subID.Valid { det.IDSubsection = int(subID.Int64) }
		det.SubTitle = subTit.String   // kosong bila NULL
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
	if err != nil { return err }
	now := time.Now()

	// 0) Pastikan CMS ada (boleh soft‑delete)
	var dummy int
	if err := tx.QueryRow(`SELECT id_cms FROM CMS WHERE id_cms = ?`,
		req.IDCMS).Scan(&dummy); err != nil {
		tx.Rollback()
		return fmt.Errorf("CMS tidak ditemukan")
	}

	// 1) Update header CMS
	if _, err := tx.Exec(`
		UPDATE CMS SET id_poli=?, title=?, updated_at=? WHERE id_cms=?`,
		req.IDPoli, req.Title, now, req.IDCMS); err != nil {
		tx.Rollback(); return err
	}

	reName := regexp.MustCompile(`[^a-zA-Z0-9\s]`)

	for _, sec := range req.Sections {
		/* ---------- SECTION ---------- */
		if sec.Deleted {
			if _, err := tx.Exec(
				`UPDATE CMS_Section SET deleted_at=? WHERE id_section=?`,
				now, sec.IDSection); err != nil { tx.Rollback(); return err }
			continue
		}

		var idSection int64
		if sec.IDSection == 0 {
			res, err := tx.Exec(
				`INSERT INTO CMS_Section (id_cms,title) VALUES (?,?)`,
				req.IDCMS, sec.Title)
			if err != nil { tx.Rollback(); return err }
			idSection, _ = res.LastInsertId()
		} else {
			idSection = int64(sec.IDSection)
			if _, err := tx.Exec(
				`UPDATE CMS_Section SET title=? WHERE id_section=?`,
				sec.Title, idSection); err != nil { tx.Rollback(); return err }
		}

		/* ---------- SUBSECTION / ELEMENT ---------- */
		if len(sec.Subsections) > 0 {
			// ======== CASE: punya subseksi ==========
			for _, sub := range sec.Subsections {
				if sub.Deleted && sub.IDSubsection != 0 {
					if _, err := tx.Exec(
						`UPDATE CMS_Subsection SET deleted_at=? WHERE id_subsection=?`,
						now, sub.IDSubsection); err != nil { tx.Rollback(); return err }
					continue
				}

				var idSub int64
				if sub.IDSubsection == 0 {
					res, err := tx.Exec(
						`INSERT INTO CMS_Subsection (id_section,title) VALUES (?,?)`,
						idSection, sub.Title)
					if err != nil { tx.Rollback(); return err }
					idSub, _ = res.LastInsertId()
				} else {
					idSub = int64(sub.IDSubsection)
					if _, err := tx.Exec(
						`UPDATE CMS_Subsection SET title=? WHERE id_subsection=?`,
						sub.Title, idSub); err != nil { tx.Rollback(); return err }
				}

				// elemen di dalam subseksi
				if err := upsertElements(tx, idSection, &idSub, sub.Elements, reName, now); err != nil {
					tx.Rollback(); return err
				}
			}
		} else if len(sec.Elements) > 0 {
			// ======== CASE: elemen langsung di section ==========
			if err := upsertElements(tx, idSection, nil, sec.Elements, reName, now); err != nil {
				tx.Rollback(); return err
			}
		}
	}

	// 2) Audit trail
	if _, err := tx.Exec(`
		INSERT INTO Management_CMS (id_management,id_cms,created_by,updated_by)
		VALUES (?,?,?,?)
		ON DUPLICATE KEY UPDATE updated_by=?`,
		mgmt.IDManagement, req.IDCMS, mgmt.CreatedBy, mgmt.UpdatedBy,
		mgmt.UpdatedBy); err != nil { tx.Rollback(); return err }

	return tx.Commit()
}

/* ---------- helper ---------- */

// upsertElements menangani create / update / soft‑delete elemen
func upsertElements(
	tx *sql.Tx,
	idSection int64,
	idSub *int64, // nil ➜ NULL di DB
	elements []models.ElementUpdate,
	reName *regexp.Regexp,
	now time.Time,
) error {

	for _, el := range elements {

		if el.Deleted && el.IDCMSElements != 0 {
			// soft‑delete element
			if _, err := tx.Exec(
				`UPDATE CMS_Elements SET deleted_at=? WHERE id_cms_elements=?`,
				now, el.IDCMSElements); err != nil {
				return err
			}
			continue
		}

		// validasi id_element
		var tmp int
		if err := tx.QueryRow(
			`SELECT id_element FROM Elements WHERE id_element=?`,
			el.IDElement).Scan(&tmp); err != nil {
			return fmt.Errorf("invalid element ID: %d", el.IDElement)
		}

		// sanitize
		clean := strings.TrimSpace(reName.ReplaceAllString(el.ElementLabel, ""))
		elemName := strings.ToLower(strings.ReplaceAll(clean, " ", "_"))

		// options
		var opts interface{}
		if len(el.ElementOptions) != 0 && string(el.ElementOptions) != "null" {
			opts = el.ElementOptions
		}

		// id_subsection NULL?
		var subID interface{} = nil
		if idSub != nil { subID = *idSub }

		if el.IDCMSElements == 0 {
			/* --------- CREATE --------- */
			rE, err := tx.Exec(`
				INSERT INTO CMS_Elements
				  (id_section,id_subsection,element_label,element_name,
				   element_options,element_hint,is_required)
				VALUES (?,?,?,?,?,?,?)`,
				idSection, subID, clean, elemName,
				opts, el.ElementHint, el.IsRequired)
			if err != nil { return err }

			newID, _ := rE.LastInsertId()
			if _, err := tx.Exec(
				`INSERT INTO Detail_Element (id_element,id_cms_elements) VALUES (?,?)`,
				el.IDElement, newID); err != nil { return err }

		} else {
			/* --------- UPDATE --------- */
			if _, err := tx.Exec(`
				UPDATE CMS_Elements
				   SET element_label=?, element_name=?, element_options=?,
				       element_hint=?, is_required=?, id_subsection=?
				 WHERE id_cms_elements=?`,
				clean, elemName, opts, el.ElementHint, el.IsRequired,
				subID, el.IDCMSElements); err != nil { return err }

			// update Detail_Element bila id_element berubah
			if _, err := tx.Exec(
				`UPDATE Detail_Element SET id_element=? WHERE id_cms_elements=?`,
				el.IDElement, el.IDCMSElements); err != nil { return err }
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
	idAntrian, idCMS, idKaryawan int,
	input models.AssessmentInput,
) (int64, error) {

	// ---------- 1. Validasi Antrian & CMS ----------
	var idPasien, idPoli int
	if err := svc.DB.QueryRow(`
		SELECT id_pasien,id_poli FROM Antrian WHERE id_antrian=?`,
		idAntrian).Scan(&idPasien, &idPoli); err != nil {

		if err == sql.ErrNoRows { return 0, ErrAntrianNotFound }
		return 0, err
	}

	var cmsPoli int
	if err := svc.DB.QueryRow(`
		SELECT id_poli FROM CMS WHERE id_cms=?`,
		idCMS).Scan(&cmsPoli); err != nil {

		if err == sql.ErrNoRows { return 0, ErrCMSNotFound }
		return 0, err
	}
	// Opsional tetap: pakai poli dari CMS
	idPoli = cmsPoli

	// ---------- 2. Ambil daftar elemen aktif ----------
	rows, err := svc.DB.Query(`
    SELECT e.element_name, e.is_required
    FROM CMS_Section s
      JOIN CMS_Elements   e  ON e.id_section   = s.id_section
                              AND e.deleted_at IS NULL
      LEFT JOIN CMS_Subsection ss ON ss.id_subsection = e.id_subsection
                                    AND (ss.deleted_at IS NULL OR e.id_subsection IS NULL)
    WHERE s.id_cms = ?                -- ← gunakan kolom di CMS_Section
      AND s.deleted_at IS NULL`,
    idCMS)

	if err != nil { return 0, err }
	defer rows.Close()

	requiredSet := map[string]struct{}{}
	allowedSet  := map[string]struct{}{}

	for rows.Next() {
		var name string
		var req  int
		if err := rows.Scan(&name, &req); err != nil { return 0, err }
		allowedSet[name] = struct{}{}
		if req != 0 { requiredSet[name] = struct{}{} }
	}

	if len(allowedSet) == 0 {
		return 0, fmt.Errorf("CMS has no active elements")
	}

	// ---------- 3. Index jawaban dari frontend ----------
	values := map[string]interface{}{}
	for _, ans := range input.Answers {
		values[ans.Name] = ans.Value
	}

	// ---------- 4. Validasi kunci tidak dikenal ----------
	var unknown []string
	for name := range values {
		if _, ok := allowedSet[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		return 0, fmt.Errorf("unknown field(s): %v", unknown)
	}

	// ---------- 5. Validasi nilai wajib terisi ----------
	var missing []string
	for name := range requiredSet {
		v, ok := values[name]
		if !ok || isEmpty(v) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return 0, fmt.Errorf("required field(s) empty: %v", missing)
	}

	// ---------- 6. Ekstrak id_ruang & id_icd10 ----------
	var (
		idRuang sql.NullInt64
		idICD10 sql.NullString
	)
	if v, ok := values["ruang_poli"].(float64); ok { // numeric dari JSON
		idRuang = sql.NullInt64{Int64: int64(v), Valid: true}
	}
	if v, ok := values["diagnosis_awal_medis"].(string); ok && v != "" {
		idICD10 = sql.NullString{String: v, Valid: true}
	}

	// ---------- 7. Simpan ----------
	raw, err := json.Marshal(input.Answers)
	if err != nil { return 0, err }

	// Cegah duplikasi (opsional): hapus assessment lama utk antrian‑CMS
	if _, err := svc.DB.Exec(`
		DELETE FROM Assessment WHERE id_antrian=? AND id_cms=?`, idAntrian, idCMS); err != nil {
		return 0, err
	}

	res, err := svc.DB.Exec(`
		INSERT INTO Assessment
		  (id_antrian,id_pasien,id_karyawan,id_poli,id_ruang,
		   id_cms,id_icd10,hasil_assessment,created_at)
		VALUES (?,?,?,?,?,?,?,?,NOW())`,
		idAntrian, idPasien, idKaryawan, idPoli, idRuang,
		idCMS, idICD10, raw)
	if err != nil { return 0, err }

	return res.LastInsertId()
}

/* ------- helper ------- */

// isEmpty menentukan "kosong" utk tipe value fleksibel
func isEmpty(v interface{}) bool {
	switch vv := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(vv) == ""
	case float64: // JSON angka
		return false // 0 dianggap terisi
	case bool:
		return false
	default:
		return false
	}
}


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