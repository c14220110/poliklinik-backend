package models

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

// ManagementCMS mewakili data dari tabel Management_CMS.
type ManagementCMS struct {
    IDManagement int    `json:"id_management"`
    CreatedBy    string `json:"created_by"`
    UpdatedBy    string `json:"updated_by"`
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