package models

import "encoding/json"

// CreateCMSRequest adalah struktur payload untuk input CMS
type CreateCMSRequest struct {
	IDPoli   int              `json:"id_poli"`
	Title    string           `json:"title"`
	Sections []SectionRequest `json:"sections"`
}

type SectionRequest struct {
	Title       string              `json:"title"`
	Subsections []SubsectionRequest `json:"subsections"`
	Elements    []ElementRequest    `json:"elements"` // jika di-level section tanpa subsection
}

type SubsectionRequest struct {
	Title    string           `json:"title"`
	Elements []ElementRequest `json:"elements"`
}

type ElementRequest struct {
	IDElement      int             `json:"id_element"`
	ElementLabel   string          `json:"element_label"`
	ElementName    string          `json:"element_name"`
	ElementOptions json.RawMessage `json:"element_options"` // bisa null atau JSON array
	ElementHint    string          `json:"element_hint"`
	IsRequired     bool            `json:"is_required"`
}

// ManagementCMS mewakili data dari tabel Management_CMS.
type ManagementCMS struct {
	IDManagement int    // fk ke Management.id_management
	CreatedBy    int    // id user
	UpdatedBy    int    // id user
}

// CMS mewakili record di tabel CMS.
type CMS struct {
    IDCMS    int    `json:"id_cms"`
    IDPoli   int    `json:"id_poli"`
    Title    string `json:"title"`
    // created_at, updated_at dikelola oleh DB
}

// CMSElement mewakili record di tabel CMS_Elements.
type CMSElement struct {
    SectionName    string `json:"section_name"`
    SubSectionName string `json:"sub_section_name"` // Kolom baru
    IDEelements    int    `json:"id_elements"`
    IDCMS          int    `json:"id_cms"`
    ElementType    string `json:"element_type"`
    ElementLabel   string `json:"element_label"`
    ElementName    string `json:"element_name"`
    ElementOptions string `json:"element_options"` // Bisa kosong atau NULL
    ElementSize    string `json:"element_size"`    // Kolom baru
    ElementHint    string `json:"element_hint"`    // Kolom baru
    IsRequired     bool   `json:"is_required"`     // Tetap bool, default false
}

// Untuk output, kita definisikan tipe-tipe berikut:

type CMSResponse struct {
    IDCMS      int            `json:"id_cms"`
    Title      string         `json:"title"`
    CreatedAt  string         `json:"created_at"`
    Management ManagementInfo `json:"management"`
    Elements   []ElementInfo  `json:"elements"`
}

type ManagementInfo struct {
    IDManagement int    `json:"id_management"`
    CreatedBy    string `json:"created_by"`
    UpdatedBy    string `json:"updated_by"`
}

// ElementInfo untuk output response
type ElementInfo struct {
    SectionName     string `json:"section_name"`
    SubSectionName  string `json:"sub_section_name"` // Kolom baru
    IDEelements     int    `json:"id_elements"`
    ElementType     string `json:"element_type"`
    ElementLabel    string `json:"element_label"`
    ElementName     string `json:"element_name"`
    ElementOptions  string `json:"element_options"`
    ElementSize     string `json:"element_size"`
    ElementHint     string `json:"element_hint"`
    IsRequired      bool   `json:"is_required"`
}

type CMSGroup struct {
    IDPoli   int           `json:"id_poli"`
    NamaPoli string        `json:"nama_poli"`
    CMS      []CMSResponse `json:"cms"`
}

type CMSFlat struct {
    IDPoli   int    `json:"id_poli"`
    NamaPoli string `json:"nama_poli"`
    IDCms    *int   `json:"id_cms"` // Pointer untuk mendukung nilai null
}

// Elemen dalam satu CMS (lengkap dgn section/subsection & type)
type CMSElementDetail struct {
    IDCMSElement int    `json:"id_cms_elements"`
    IDSection    int    `json:"id_section"`
    SectionTitle string `json:"section_title"`
    IDSubsection int    `json:"id_subsection"`
    SubTitle     string `json:"subsection_title"`
    IDElement    int    `json:"id_element"`   // master Elements
    ElementType  string `json:"element_type"` // dari tabel Elements.type
    Label        string `json:"label"`
    Name         string `json:"name"`
    Options      string `json:"options"` // raw JSON/text
    Hint         string `json:"hint"`
    Required     bool   `json:"required"`
}

type CMSDetailResponse struct {
    IDCMS    int                `json:"id_cms"`
    Title    string             `json:"title"`
    Elements []CMSElementDetail `json:"elements"`
}

type CMSListItem struct {
	IDCMS  int    `json:"id_cms"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

////apdet
type UpdateCMSRequest struct {
	IDCMS   int               `json:"id_cms"`                 // wajib
	IDPoli  int               `json:"id_poli"`                // boleh di-update
	Title   string            `json:"title"`                  // boleh di-update
	Sections []SectionUpdate  `json:"sections"`               // state lengkap
}

// ---------- helper struct ----------
type SectionUpdate struct {
	IDSection   int                `json:"id_section,omitempty"`   // 0 = section baru
	Title       string             `json:"title"`
	Deleted     bool               `json:"deleted,omitempty"`      // true = hapus section
	Subsections []SubsectionUpdate `json:"subsections,omitempty"`  // opsional
	Elements    []ElementUpdate    `json:"elements,omitempty"`     // section tanpa subsection
}

type SubsectionUpdate struct {
	IDSubsection int             `json:"id_subsection,omitempty"` // 0 = baru
	Title        string          `json:"title"`
	Deleted      bool            `json:"deleted,omitempty"`
	Elements     []ElementUpdate `json:"elements,omitempty"`
}

type ElementUpdate struct {
	IDCMSElements int             `json:"id_cms_elements,omitempty"` // 0 = baru
	Deleted       bool            `json:"deleted,omitempty"`
	IDElement     int             `json:"id_element"`                // foreign-key master Elements
	ElementLabel  string          `json:"element_label"`
	ElementOptions json.RawMessage `json:"element_options"`
	ElementHint   string          `json:"element_hint"`
	IsRequired    bool            `json:"is_required"`
}

// satu jawaban elemen
type CMSAnswer struct {
    IDCmsElement int    `json:"id_cms_elements"`
    Label        string `json:"label"`
    Name         string `json:"name"`   // <-- tambahkan
    Value        any    `json:"value"`
}


// payload dari frontend
type AssessmentInput struct {
    Answers []CMSAnswer `json:"answers"`
}