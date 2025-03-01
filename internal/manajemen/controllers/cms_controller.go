package controllers

import (
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/models"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/c14220110/poliklinik-backend/pkg/utils"
	"github.com/labstack/echo/v4"
)

type CMSController struct {
    Service *services.CMSService
}

func NewCMSController(service *services.CMSService) *CMSController {
    return &CMSController{Service: service}
}

// CreateCMSRequest adalah struktur payload untuk input CMS (tanpa id_poli).
type CreateCMSRequest struct {
    Title    string              `json:"title"`
    Elements []CMSElementRequest `json:"elements"`
}

type CMSElementRequest struct {
    ElementType    string `json:"element_type"`
    ElementLabel   string `json:"element_label"`
    ElementName    string `json:"element_name"`
    ElementOptions string `json:"element_options"`
    IsRequired     bool   `json:"is_required"`
}

func (cc *CMSController) CreateCMSHandler(c echo.Context) error {
    // Ambil id_poli dari query parameter
    idPoliStr := c.QueryParam("id_poli")
    if idPoliStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_poli query parameter is required",
            "data":    nil,
        })
    }
    idPoli, err := strconv.Atoi(idPoliStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_poli must be a valid number",
            "data":    nil,
        })
    }

    // Ambil klaim dari token (management)
    claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
    if !ok || claims == nil {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid or missing token claims",
            "data":    nil,
        })
    }

    var req CreateCMSRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "Invalid request payload: " + err.Error(),
            "data":    nil,
        })
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

    // Ambil id_management dari token JWT
    idManagement, err := strconv.Atoi(claims.IDKaryawan)
    if err != nil || idManagement <= 0 {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid management ID in token",
            "data":    nil,
        })
    }

    // Buat objek ManagementCMS
    managementInfo := models.ManagementCMS{
        IDManagement: idManagement,
        CreatedBy:    claims.Username,
        UpdatedBy:    claims.Username,
    }

    // Panggil service untuk membuat CMS dengan elemen
    idCMS, err := cc.Service.CreateCMSWithElements(cms, elements, managementInfo)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to create CMS: " + err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "CMS created successfully",
        "data": map[string]interface{}{
            "id_cms": idCMS,
        },
    })
}

func (cc *CMSController) GetCMSByPoliklinikHandler(c echo.Context) error {
    poliIDStr := c.QueryParam("id_poli")
    if poliIDStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_poli parameter is required",
            "data":    nil,
        })
    }
    poliID, err := strconv.Atoi(poliIDStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_poli must be a number",
            "data":    nil,
        })
    }

    cmsList, err := cc.Service.GetCMSByPoliklinikID(poliID)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve CMS data: " + err.Error(),
            "data":    nil,
        })
    }

    response := map[string]interface{}{
        "id_poli": poliID,
        "cms":     cmsList,
    }
    return c.JSON(http.StatusOK, response)
}

func (cc *CMSController) GetAllCMSHandler(c echo.Context) error {
    groups, err := cc.Service.GetAllCMS()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve all CMS data: " + err.Error(),
            "data":    nil,
        })
    }
    response := map[string]interface{}{
        "cms_by_poliklinik": groups,
    }
    return c.JSON(http.StatusOK, response)
}

func (cc *CMSController) UpdateCMSHandler(c echo.Context) error {
    idCMSStr := c.QueryParam("id_cms")
    if idCMSStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_cms parameter is required",
            "data":    nil,
        })
    }
    idCMS, err := strconv.Atoi(idCMSStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_cms must be a number",
            "data":    nil,
        })
    }

    // Ambil payload JSON untuk update CMS
    type UpdateCMSRequest struct {
        Title    string              `json:"title"`
        Elements []CMSElementRequest `json:"elements"`
    }
    var req UpdateCMSRequest
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "Invalid request payload: " + err.Error(),
            "data":    nil,
        })
    }

    // Ambil klaim JWT dari context
    claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
    if !ok || claims == nil {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid or missing token claims",
            "data":    nil,
        })
    }

    // Ambil id_management dari token
    idManagement, err := strconv.Atoi(claims.IDKaryawan)
    if err != nil || idManagement <= 0 {
        return c.JSON(http.StatusUnauthorized, map[string]interface{}{
            "status":  http.StatusUnauthorized,
            "message": "Invalid management ID in token",
            "data":    nil,
        })
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

    // Buat objek ManagementCMS untuk update
    managementInfo := models.ManagementCMS{
        IDManagement: idManagement,
        UpdatedBy:    claims.Username,
    }

    // Panggil service untuk update CMS
    err = cc.Service.UpdateCMSWithElements(idCMS, req.Title, elements, managementInfo)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to update CMS: " + err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "CMS updated successfully",
        "data": map[string]interface{}{
            "id_cms": idCMS,
        },
    })
}