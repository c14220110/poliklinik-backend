package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
)

type CMSController struct {
	Service *services.CMSService
}

func NewCMSController(service *services.CMSService) *CMSController {
	return &CMSController{Service: service}
}

// CreateCMSRequest adalah struktur payload untuk input CMS (tanpa id_poli).
type CreateCMSRequest struct {
	Title    string               `json:"title"`
	Elements []CMSElementRequest  `json:"elements"`
}

type CMSElementRequest struct {
	ElementType    string `json:"element_type"`
	ElementLabel   string `json:"element_label"`
	ElementName    string `json:"element_name"`
	ElementOptions string `json:"element_options"`
	IsRequired     bool   `json:"is_required"`
}

// CreateCMSHandler menangani endpoint POST untuk input CMS.
// id_poli diambil dari query parameter; informasi management diambil dari token JWT.
func (cc *CMSController) CreateCMSHandler(w http.ResponseWriter, r *http.Request) {
	// Ambil id_poli dari query parameter
	idPoliStr := r.URL.Query().Get("id_poli")
	if idPoliStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli query parameter is required",
			"data":    nil,
		})
		return
	}
	idPoli, err := strconv.Atoi(idPoliStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli must be a valid number",
			"data":    nil,
		})
		return
	}

	// Ambil klaim dari token (management)
	claims, ok := r.Context().Value(middlewares.ContextKeyClaims).(*utils.Claims)
	if !ok || claims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
		return
	}

	var req CreateCMSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Buat objek CMS
	cms := models.CMS{
		IDPoli: idPoli,
		Title:  req.Title,
	}

	// Konversi elemen request ke models.CMSElement
	var elements []models.CMSElement
	for _, e := range req.Elements {
		elem := models.CMSElement{
			ElementType:    e.ElementType,
			ElementLabel:   e.ElementLabel,
			ElementName:    e.ElementName,
			ElementOptions: e.ElementOptions,
			IsRequired:     e.IsRequired,
		}
		elements = append(elements, elem)
	}

	// Ambil id_management dari token JWT (asumsikan klaim "id_karyawan" menyimpan id_management sebagai string)
	idManagement, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idManagement <= 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid management ID in token",
			"data":    nil,
		})
		return
	}

	// Buat objek ManagementCMS menggunakan username dari token untuk created_by dan updated_by
	managementInfo := models.ManagementCMS{
		IDManagement: idManagement,
		CreatedBy:    claims.Username,
		UpdatedBy:    claims.Username,
	}

	// Panggil service untuk membuat CMS dengan elemen dan mencatat di Management_CMS
	idCMS, err := cc.Service.CreateCMSWithElements(cms, elements, managementInfo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to create CMS: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "CMS created successfully",
		"data": map[string]interface{}{
			"id_cms": idCMS,
		},
	})
}



// GetCMSByPoliklinikHandler mengembalikan data CMS untuk poliklinik tertentu.
func (cc *CMSController) GetCMSByPoliklinikHandler(w http.ResponseWriter, r *http.Request) {
	poliIDStr := r.URL.Query().Get("id_poli") // gunakan "id_poli" di sini
	if poliIDStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli parameter is required",
			"data":    nil,
		})
		return
	}
	poliID, err := strconv.Atoi(poliIDStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_poli must be a number",
			"data":    nil,
		})
		return
	}

	cmsList, err := cc.Service.GetCMSByPoliklinikID(poliID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve CMS data: " + err.Error(),
			"data":    nil,
		})
		return
	}

	response := map[string]interface{}{
		"id_poli": poliID,
		"cms":     cmsList,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}


// GetAllCMSHandler mengembalikan semua CMS yang dikelompokkan berdasarkan poliklinik.
func (cc *CMSController) GetAllCMSHandler(w http.ResponseWriter, r *http.Request) {
	groups, err := cc.Service.GetAllCMS()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve all CMS data: " + err.Error(),
			"data":    nil,
		})
		return
	}
	response := map[string]interface{}{
		"cms_by_poliklinik": groups,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// UpdateCMSHandler menangani endpoint PUT untuk update CMS.
// URL: /api/cms/update?id_cms=5
func (cc *CMSController) UpdateCMSHandler(w http.ResponseWriter, r *http.Request) {
	// Ambil query parameter id_cms
	idCMSStr := r.URL.Query().Get("id_cms")
	if idCMSStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_cms parameter is required",
			"data":    nil,
		})
		return
	}
	idCMS, err := strconv.Atoi(idCMSStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_cms must be a number",
			"data":    nil,
		})
		return
	}

	// Ambil payload JSON untuk update CMS
	type UpdateCMSRequest struct {
		Title    string              `json:"title"`
		Elements []CMSElementRequest `json:"elements"`
	}
	var req UpdateCMSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request payload: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Ambil klaim JWT dari context
	claims, ok := r.Context().Value(middlewares.ContextKeyClaims).(*utils.Claims)
	if !ok || claims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
		return
	}

	// Ambil id_management dari token (dari klaim "id_karyawan")
	idManagement, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || idManagement <= 0 {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusUnauthorized,
			"message": "Invalid management ID in token",
			"data":    nil,
		})
		return
	}

	// Konversi elemen dari request ke models.CMSElement
	var elements []models.CMSElement
	for _, e := range req.Elements {
		elem := models.CMSElement{
			ElementType:    e.ElementType,
			ElementLabel:   e.ElementLabel,
			ElementName:    e.ElementName,
			ElementOptions: e.ElementOptions,
			IsRequired:     e.IsRequired,
		}
		elements = append(elements, elem)
	}

	// Buat objek ManagementCMS untuk update (gunakan username dari token untuk updated_by)
	managementInfo := models.ManagementCMS{
		IDManagement: idManagement,
		UpdatedBy:    claims.Username,
	}

	// Panggil service untuk update CMS dengan elemen baru
	err = cc.Service.UpdateCMSWithElements(idCMS, req.Title, elements, managementInfo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to update CMS: " + err.Error(),
			"data":    nil,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  http.StatusOK,
		"message": "CMS updated successfully",
		"data": map[string]interface{}{
			"id_cms": idCMS,
		},
	})
}
