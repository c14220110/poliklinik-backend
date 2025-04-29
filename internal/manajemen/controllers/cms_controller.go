package controllers

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	//"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
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
// CreateCMSHandler handles POST /api/management/cms
func (cc *CMSController) CreateCMSHandler(c echo.Context) error {
	// bind request
	var req models.CreateCMSRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"status": http.StatusBadRequest, "message": "Invalid payload: " + err.Error()})
	}
	if req.IDPoli == 0 || req.Title == "" || len(req.Sections) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"status": http.StatusBadRequest, "message": "id_poli, title, sections wajib diisi"})
	}

	// sanitize and generate element names
	re := regexp.MustCompile(`[^a-zA-Z0-9\s]`)
	for si, sec := range req.Sections {
		for ei, el := range sec.Elements {
			clean := strings.TrimSpace(re.ReplaceAllString(el.ElementLabel, ""))
			name := strings.ToLower(strings.ReplaceAll(clean, " ", "_"))
			req.Sections[si].Elements[ei].ElementLabel = clean
			req.Sections[si].Elements[ei].ElementName = name
		}
		for sbi, sub := range sec.Subsections {
			for ei, el := range sub.Elements {
				clean := strings.TrimSpace(re.ReplaceAllString(el.ElementLabel, ""))
				name := strings.ToLower(strings.ReplaceAll(clean, " ", "_"))
				req.Sections[si].Subsections[sbi].Elements[ei].ElementLabel = clean
				req.Sections[si].Subsections[sbi].Elements[ei].ElementName = name
			}
		}
	}

	// get userID
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"status": http.StatusUnauthorized, "message": "Invalid token claims"})
	}
	uid, err := strconv.Atoi(claims.IDKaryawan)
	if err != nil || uid <= 0 {
		return c.JSON(http.StatusUnauthorized, echo.Map{"status": http.StatusUnauthorized, "message": "Invalid user ID"})
	}
	mgmt := models.ManagementCMS{IDManagement: uid, CreatedBy: uid, UpdatedBy: uid}

	// call service
	idCMS, err := cc.Service.CreateCMSWithSections(req, mgmt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"status": http.StatusInternalServerError, "message": "Failed to create CMS: " + err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"status": http.StatusOK, "message": "CMS created successfully", "data": map[string]int{"id_cms": int(idCMS)}})
}


// GetCMSDetailHandler handles GET /api/management/cms?id_cms={id}
func (cc *CMSController) GetCMSDetailHandler(c echo.Context) error {
	// parse id_cms
	idStr := c.QueryParam("id_cms")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_cms parameter is required",
			"data":    nil,
		})
	}
	cmsID, err := strconv.Atoi(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_cms must be a number",
			"data":    nil,
		})
	}

	// call service
	resp, err := cc.Service.GetCMSDetailByID(cmsID)
	if err != nil {
		if err == services.ErrCMSNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"status":  http.StatusNotFound,
				"message": "CMS not found",
				"data":    nil,
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve CMS detail: " + err.Error(),
			"data":    nil,
		})
	}

	// success response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "CMS detail retrieved successfully",
		"data":    resp,
	})
}



// GetCMSListByPoliHandler handles GET /api/management/cms/list?id_poli={id}
func (cc *CMSController) GetCMSListByPoliHandler(c echo.Context) error {
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
            "message": "id_poli must be a valid number",
            "data":    nil,
        })
    }

    list, err := cc.Service.GetCMSListByPoli(poliID)
    if err != nil {
        if err == services.ErrNoCMSForPoli {
            return c.JSON(http.StatusNotFound, map[string]interface{}{
                "status":  http.StatusNotFound,
                "message": "No CMS entries found for this poliklinik",
                "data":    nil,
            })
        }
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve CMS list: " + err.Error(),
            "data":    nil,
        })
    }

    // wrap id_poli and list into data
    payload := map[string]interface{}{
        "id_poli": poliID,
        "cms":     list,
    }
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "CMS list retrieved successfully",
        "data":    payload,
    })
}


// controllers/cms_controller.go
func (cc *CMSController) UpdateCMSHandler(c echo.Context) error {
	var req models.UpdateCMSRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status": http.StatusBadRequest, "message": "Invalid payload: "+err.Error()})
	}
	if req.IDCMS == 0 || req.IDPoli == 0 || len(req.Sections) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status": http.StatusBadRequest, "message": "id_cms, id_poli, sections wajib diisi"})
	}

	// ambil user ID dari JWT (sama seperti handler POST)
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"status": http.StatusUnauthorized, "message": "Invalid token claims"})
	}
	uid, _ := strconv.Atoi(claims.IDKaryawan)
	mgmt := models.ManagementCMS{IDManagement: uid, CreatedBy: uid, UpdatedBy: uid}

	if err := cc.Service.UpdateCMSWithSections(req, mgmt); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"status": http.StatusInternalServerError, "message": "Failed to update CMS: "+err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{
		"status": http.StatusOK, "message": "CMS updated successfully"})
}

// PUT /api/management/cms/activate?id_cms={id}
func (cc *CMSController) ActivateCMSHandler(c echo.Context) error {
    idStr := c.QueryParam("id_cms")
    if idStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_cms parameter is required",
            "data":    nil,
        })
    }
    cmsID, err := strconv.Atoi(idStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_cms must be a number",
            "data":    nil,
        })
    }

    // call service
    err = cc.Service.ActivateCMS(cmsID)
    if err != nil {
        switch err {
        case services.ErrCMSNotFound:
            return c.JSON(http.StatusNotFound, map[string]interface{}{
                "status":  http.StatusNotFound,
                "message": "CMS not found",
                "data":    nil,
            })
        case services.ErrCMSAlreadyActive:
            return c.JSON(http.StatusBadRequest, map[string]interface{}{
                "status":  http.StatusBadRequest,
                "message": "CMS is already active",
                "data":    nil,
            })
        case services.ErrOtherCMSActive:
            return c.JSON(http.StatusConflict, map[string]interface{}{
                "status":  http.StatusConflict,
                "message": "Another active CMS exists for this poliklinik",
                "data":    nil,
            })
        default:
            return c.JSON(http.StatusInternalServerError, map[string]interface{}{
                "status":  http.StatusInternalServerError,
                "message": "Failed to activate CMS: " + err.Error(),
                "data":    nil,
            })
        }
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "CMS activated successfully",
        "data":    map[string]interface{}{"id_cms": cmsID},
    })
}


// PUT /api/management/cms/deactivate?id_cms={id}
func (cc *CMSController) DeactivateCMSHandler(c echo.Context) error {
    idStr := c.QueryParam("id_cms")
    if idStr=="" { return c.JSON(http.StatusBadRequest,map[string]interface{}{ "status":http.StatusBadRequest,"message":"id_cms parameter is required","data":nil}) }
    id,err:=strconv.Atoi(idStr); if err!=nil { return c.JSON(http.StatusBadRequest,map[string]interface{}{ "status":http.StatusBadRequest,"message":"id_cms must be a number","data":nil}) }
    err = cc.Service.DeactivateCMS(id)
    if err!=nil {
        switch err {
        case services.ErrCMSNotFound:
            return c.JSON(http.StatusNotFound,map[string]interface{}{ "status":http.StatusNotFound,"message":"CMS not found","data":nil})
        case services.ErrCMSNotActive:
            return c.JSON(http.StatusBadRequest,map[string]interface{}{ "status":http.StatusBadRequest,"message":"CMS is already non‑active","data":nil})
        default:
            return c.JSON(http.StatusInternalServerError,map[string]interface{}{ "status":http.StatusInternalServerError,"message":"Failed to deactivate CMS: "+err.Error(),"data":nil})
        }
    }
    return c.JSON(http.StatusOK,map[string]interface{}{ "status":http.StatusOK,"message":"CMS deactivated successfully","data":map[string]interface{}{"id_cms":id}})
}
