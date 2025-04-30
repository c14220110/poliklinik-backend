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



// GetCMSDetailFull mengembalikan detail CMS beserta elemen lengkap
func (svc *CMSService) GetCMSDetailFull(cmsID int) (models.CMSDetailResponse, error) {
    var resp models.CMSDetailResponse

    // Ambil header CMS (boleh aktif / non-aktif)
    err := svc.DB.QueryRow(`SELECT id_cms, title FROM CMS WHERE id_cms = ?`, cmsID).
        Scan(&resp.IDCMS, &resp.Title)
    if err != nil {
        if err == sql.ErrNoRows {
            return resp, ErrCMSNotFound
        }
        return resp, err
    }

    // Join ke semua tabel pendukung
    query := `
      SELECT
        e.id_cms_elements,
        s.id_section,       s.title              AS section_title,
        ss.id_subsection,   ss.title             AS subsection_title,
        d.id_element,       el.type              AS element_type,
        e.element_label,    e.element_name,
        COALESCE(e.element_options,'') AS options,
        COALESCE(e.element_hint,'')    AS hint,
        e.is_required
      FROM CMS_Section s
      JOIN CMS_Subsection ss ON ss.id_section = s.id_section
      JOIN CMS_Elements   e  ON e.id_section  = s.id_section
                             AND e.id_subsection = ss.id_subsection
      JOIN Detail_Element d  ON d.id_cms_elements = e.id_cms_elements
      JOIN Elements       el ON el.id_element = d.id_element
      WHERE s.id_cms = ?
      ORDER BY s.id_section, ss.id_subsection, e.id_cms_elements
    `
    rows, err := svc.DB.Query(query, cmsID)
    if err != nil {
        return resp, err
    }
    defer rows.Close()

    for rows.Next() {
        var (
            det   models.CMSElementDetail
            req   int
        )
        if err := rows.Scan(
            &det.IDCMSElement,
            &det.IDSection, &det.SectionTitle,
            &det.IDSubsection, &det.SubTitle,
            &det.IDElement, &det.ElementType,
            &det.Label, &det.Name,
            &det.Options, &det.Hint,
            &req,
        ); err != nil {
            return resp, err
        }
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

var ErrAntrianNotFound = errors.New("antrian not found")

// SaveAssessment menyimpan jawaban CMS untuk satu antrian.
func (svc *CMSService) SaveAssessment(
    idAntrian, idCMS, idKaryawan int,
    input models.AssessmentInput,
) (int64, error) {

    // 1. Ambil id_pasien & id_poli dari Antrian + validasi
    var idPasien, idPoli int
    err := svc.DB.QueryRow(
        "SELECT id_pasien, id_poli FROM Antrian WHERE id_antrian = ?", idAntrian,
    ).Scan(&idPasien, &idPoli)
    if err != nil {
        if err == sql.ErrNoRows { return 0, ErrAntrianNotFound }
        return 0, err
    }

    // 2. Pastikan id_cms belong to that poli
    var cmsPoli int
    if err := svc.DB.QueryRow(
        "SELECT id_poli FROM CMS WHERE id_cms = ?", idCMS,
    ).Scan(&cmsPoli); err != nil {
        if err == sql.ErrNoRows { return 0, ErrCMSNotFound }
        return 0, err
    }

    // (opsional) boleh lintaskan poli berbeda, tapi ikuti permintaan:
    idPoli = cmsPoli

    // 3. Tarik id_ruang & id_icd10 dari answers
    var (
        idRuang  sql.NullInt64
        idICD10  sql.NullString
    )
    for _, a := range input.Answers {
        switch a.Name {          // GAJADI switch a.Label
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

    // 4. Marshal answers menjadi JSON
    raw, err := json.Marshal(input.Answers)
    if err != nil { return 0, err }

    // 5. Insert ke Assessment
    res, err := svc.DB.Exec(`
        INSERT INTO Assessment
          (id_pasien, id_karyawan, id_poli, id_ruang, id_cms, id_icd10, hasil_assessment, created_at)
        VALUES (?,?,?,?,?,?,?,?)
    `,
        idPasien,
        idKaryawan,
        idPoli,
        idRuang,
        idCMS,
        idICD10,
        raw,
        time.Now(),
    )
    if err != nil { return 0, err }

    return res.LastInsertId()
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