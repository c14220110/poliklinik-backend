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
	uid := claims.IDKaryawan

	mgmt := models.ManagementCMS{IDManagement: uid, CreatedBy: uid, UpdatedBy: uid}

	// call service
	idCMS, err := cc.Service.CreateCMSWithSections(req, mgmt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"status": http.StatusInternalServerError, "message": "Failed to create CMS: " + err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"status": http.StatusOK, "message": "CMS created successfully", "data": map[string]int{"id_cms": int(idCMS)}})
}


func (cc *CMSController) GetCMSDetailByPoliHandler(c echo.Context) error {
	poliIDStr := c.QueryParam("id_poli")
	if poliIDStr == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "id_poli parameter is required",
			"data":    nil,
		})
	}
	poliID, err := strconv.Atoi(poliIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "id_poli must be a number",
			"data":    nil,
		})
	}

	detail, err := cc.Service.GetActiveCMSDetailByPoliID(poliID)
	if err != nil {
		if err == services.ErrNoActiveCMSFound {
			return c.JSON(http.StatusNotFound, echo.Map{
				"status":  http.StatusNotFound,
				"message": "No active CMS found for this polyclinic",
				"data":    nil,
			})
		}
		if err == services.ErrMultipleActiveCMSFound {
			return c.JSON(http.StatusConflict, echo.Map{
				"status":  http.StatusConflict,
				"message": "Multiple active CMS found for this polyclinic",
				"data":    nil,
			})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve CMS detail: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  http.StatusOK,
		"message": "CMS detail retrieved successfully",
		"data":    detail,
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
	uid := claims.IDKaryawan
	mgmt := models.ManagementCMS{IDManagement: uid, CreatedBy: uid, UpdatedBy: uid}

	if err := cc.Service.UpdateCMSWithSections(req, mgmt); err != nil {
	return c.JSON(http.StatusInternalServerError, echo.Map{
		"status":  http.StatusInternalServerError,
		"message": "Failed to update CMS: " + err.Error(),
		"data":    nil,                    // ← konsisten
	})
}

return c.JSON(http.StatusOK, echo.Map{
	"status":  http.StatusOK,
	"message": "CMS updated successfully",
	"data":    echo.Map{"id_cms": req.IDCMS},   // ← tambahkan data
})

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


// SaveAssessmentHandler
// POST /api/dokter/assessment?id_antrian={id}&id_poli={id}
func (cc *CMSController) SaveAssessmentHandler(c echo.Context) error {
	/* ---------- query-param ---------- */
	idAntrianStr := c.QueryParam("id_antrian")
	idPoliStr    := c.QueryParam("id_poli")
	if idAntrianStr == "" || idPoliStr == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "id_antrian & id_poli are required",
			"data":    nil,
		})
	}

	idAntrian, err1 := strconv.Atoi(idAntrianStr)
	idPoli,    err2 := strconv.Atoi(idPoliStr)
	if err1 != nil || err2 != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "id_antrian & id_poli must be numbers",
			"data":    nil,
		})
	}

	/* ---------- JWT (dokter) ---------- */
	claims, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"status":  http.StatusUnauthorized,
			"message": "Invalid token claims",
			"data":    nil,
		})
	}
	idKaryawan := claims.IDKaryawan

	/* ---------- payload ---------- */
	var input models.AssessmentInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "Invalid JSON payload: " + err.Error(),
			"data":    nil,
		})
	}

	/* ---------- service ---------- */
	idAss, err := cc.Service.SaveAssessment(idAntrian, idPoli, idKaryawan, input)
	if err != nil {
		switch err {
		case services.ErrAntrianNotFound:
			return c.JSON(http.StatusNotFound, echo.Map{"status": http.StatusNotFound, "message": "Antrian not found"})
		case services.ErrCMSMultipleActive:
			return c.JSON(http.StatusConflict, echo.Map{"status": http.StatusConflict, "message": "Poliklinik memiliki lebih dari 1 CMS aktif"})
		case services.ErrCMSNoneActive:
			return c.JSON(http.StatusNotFound, echo.Map{"status": http.StatusNotFound, "message": "Poliklinik tidak memiliki CMS aktif"})
		case services.ErrCMSNeverCreated:
			return c.JSON(http.StatusNotFound, echo.Map{"status": http.StatusNotFound, "message": "Poliklinik belum memiliki CMS sama sekali"})
		default:
			// validasi custom → 400
			if strings.HasPrefix(err.Error(), "unknown id_cms_elements") ||
				strings.HasPrefix(err.Error(), "required id_cms_elements") {
				return c.JSON(http.StatusBadRequest, echo.Map{
					"status":  http.StatusBadRequest,
					"message": err.Error(),
					"data":    nil,
				})
			}
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"status":  http.StatusInternalServerError,
				"message": "Failed to save assessment: " + err.Error(),
				"data":    nil,
			})
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  http.StatusOK,
		"message": "Assessment saved successfully",
		"data":    echo.Map{"id_assessment": idAss},
	})
}

func (cc *CMSController) GetRincianAsesmenHandler(c echo.Context) error {
    idStr := c.QueryParam("id_antrian")
    if idStr == "" {
        return c.JSON(http.StatusBadRequest, echo.Map{
            "status":  http.StatusBadRequest,
            "message": "id_antrian parameter is required",
            "data":    nil,
        })
    }
    idAntrian, err := strconv.Atoi(idStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, echo.Map{
            "status":  http.StatusBadRequest,
            "message": "id_antrian must be a number",
            "data":    nil,
        })
    }

    rincian, err := cc.Service.GetRincianByAntrian(idAntrian)
    if err != nil {
        switch err {
        case services.ErrAntrianNotFound:
            return c.JSON(http.StatusNotFound, echo.Map{
                "status":  http.StatusNotFound,
                "message": "Antrian not found",
                "data":    nil,
            })
        case services.ErrAssessmentAbsent:
            return c.JSON(http.StatusNotFound, echo.Map{
                "status":  http.StatusNotFound,
                "message": "No assessment yet for this antrian",
                "data":    nil,
            })
        default:
            return c.JSON(http.StatusInternalServerError, echo.Map{
                "status":  http.StatusInternalServerError,
                "message": "Failed to retrieve detail: " + err.Error(),
                "data":    nil,
            })
        }
    }

    return c.JSON(http.StatusOK, echo.Map{
        "status":  http.StatusOK,
        "message": "Detail antrian retrieved successfully",
        "data":    rincian,
    })
}

func (cc *CMSController) GetAssessmentDetail(c echo.Context) error {
	idStr := c.QueryParam("id_assessment")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "id_assessment query parameter is required",
			"data":    nil,
		})
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"status":  http.StatusBadRequest,
			"message": "id_assessment must be a positive number",
			"data":    nil,
		})
	}

	// boleh di-comment kalau endpoint publik
	if _, ok := c.Get(string(middlewares.ContextKeyClaims)).(*utils.Claims); !ok {
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"status":  http.StatusUnauthorized,
			"message": "Invalid or missing token claims",
			"data":    nil,
		})
	}

	detail, err := cc.Service.GetAssessmentDetailFull(id)
	if err != nil {
		if err == services.ErrAssessmentNotFound {
			return c.JSON(http.StatusNotFound, echo.Map{
				"status":  http.StatusNotFound,
				"message": "Assessment not found",
				"data":    nil,
			})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"status":  http.StatusInternalServerError,
			"message": "Failed to retrieve assessment: " + err.Error(),
			"data":    nil,
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":  http.StatusOK,
		"message": "Assessment retrieved successfully",
		"data":    detail,
	})
}


// MoveCMS handles the PUT /move-to?id_cms endpoint to move an inactive CMS to a new poli
func (cc *CMSController) MoveCMS(ctx echo.Context) error {
	// Ambil id_cms dari query parameter
	idCMSStr := ctx.QueryParam("id_cms")
	idCMS, err := strconv.Atoi(idCMSStr)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid id_cms",
			"data":    nil,
		})
	}

	// Ambil id_poli dari body request
	var req struct {
		IDPoli int `json:"id_poli"`
	}
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "Invalid request body",
			"data":    nil,
		})
	}

	// Panggil service untuk memindahkan CMS
	err = cc.Service.MoveCMS(idCMS, req.IDPoli)
	if err != nil {
		if err == services.ErrCMSNotFound {
			return ctx.JSON(http.StatusNotFound, map[string]interface{}{
				"status":  http.StatusNotFound,
				"message": "CMS not found",
				"data":    nil,
			})
		} else if err == services.ErrCMSNotInactive {
			return ctx.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  http.StatusBadRequest,
				"message": "CMS must be inactive to be moved",
				"data":    nil,
			})
		} else {
			return ctx.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to move CMS: " + err.Error(),
				"data":    nil,
			})
		}
	}

	// Jika berhasil, kembalikan response sukses
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "CMS moved successfully",
		"data":    nil,
	})
}

func (cc *CMSController) DuplicateCMSHandler(c echo.Context) error {
	idStr := c.QueryParam("id_cms")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_cms parameter is required",
			"data":    nil,
		})
	}
	idCMS, err := strconv.Atoi(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status":  http.StatusBadRequest,
			"message": "id_cms must be a number",
			"data":    nil,
		})
	}

	newIDCMS, err := cc.Service.DuplicateCMS(idCMS)
	if err != nil {
		switch err {
		case services.ErrCMSNotFound:
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"status":  http.StatusNotFound,
				"message": "CMS not found",
				"data":    nil,
			})
		case services.ErrCMSNotInactive:
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"status":  http.StatusBadRequest,
				"message": "Only inactive CMS can be duplicated",
				"data":    nil,
			})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"status":  http.StatusInternalServerError,
				"message": "Failed to duplicate CMS: " + err.Error(),
				"data":    nil,
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  http.StatusOK,
		"message": "CMS duplicated successfully",
		"data":    map[string]interface{}{"new_id_cms": newIDCMS},
	})
}