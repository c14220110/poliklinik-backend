package controllers

import (
	"net/http"
	"strconv"

	"github.com/c14220110/poliklinik-backend/internal/manajemen/services"
	"github.com/labstack/echo/v4"
)

type RoleController struct {
    Service *services.RoleService
}

func NewRoleController(service *services.RoleService) *RoleController {
    return &RoleController{Service: service}
}

// AddRoleHandler menangani endpoint POST untuk menambah Role.
func (rc *RoleController) AddRoleHandler(c echo.Context) error {
    var req struct {
        NamaRole string `json:"nama_role"`
    }
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "Invalid request payload: " + err.Error(),
            "data":    nil,
        })
    }
    if req.NamaRole == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "nama_role is required",
            "data":    nil,
        })
    }

    id, err := rc.Service.AddRole(req.NamaRole)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Role added successfully",
        "data":    map[string]interface{}{"id_role": id},
    })
}

// UpdateRoleHandler menangani endpoint PUT untuk memperbarui Role berdasarkan query parameter id_role.
func (rc *RoleController) UpdateRoleHandler(c echo.Context) error {
    idRoleStr := c.QueryParam("id_role")
    if idRoleStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_role parameter is required",
            "data":    nil,
        })
    }
    idRole, err := strconv.Atoi(idRoleStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_role must be a number",
            "data":    nil,
        })
    }

    var req struct {
        NamaRole string `json:"nama_role"`
    }
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "Invalid request payload: " + err.Error(),
            "data":    nil,
        })
    }
    if req.NamaRole == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "nama_role is required",
            "data":    nil,
        })
    }

    err = rc.Service.UpdateRole(idRole, req.NamaRole)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Role updated successfully",
        "data":    map[string]interface{}{"id_role": idRole},
    })
}

// SoftDeleteRoleHandler menangani endpoint PUT untuk soft delete Role berdasarkan query parameter id_role.
func (rc *RoleController) SoftDeleteRoleHandler(c echo.Context) error {
    idRoleStr := c.QueryParam("id_role")
    if idRoleStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_role parameter is required",
            "data":    nil,
        })
    }
    idRole, err := strconv.Atoi(idRoleStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_role must be a number",
            "data":    nil,
        })
    }

    err = rc.Service.SoftDeleteRole(idRole)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Role soft-deleted successfully",
        "data":    map[string]interface{}{"id_role": idRole},
    })
}

// GetRoleListHandler menangani endpoint GET untuk mengambil daftar role dengan filter opsional.
func (rc *RoleController) GetRoleListHandler(c echo.Context) error {
    statusFilter := c.QueryParam("status")
    roles, err := rc.Service.GetRoleListFiltered(statusFilter)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to retrieve role list: " + err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Role list retrieved successfully",
        "data":    roles,
    })
}

// ActivateRoleHandler mengubah deleted_at menjadi NULL untuk role tertentu.
func (rc *RoleController) ActivateRoleHandler(c echo.Context) error {
    idRoleStr := c.QueryParam("id_role")
    if idRoleStr == "" {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_role parameter is required",
            "data":    nil,
        })
    }
    idRole, err := strconv.Atoi(idRoleStr)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]interface{}{
            "status":  http.StatusBadRequest,
            "message": "id_role must be a number",
            "data":    nil,
        })
    }

    err = rc.Service.ActivateRole(idRole)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]interface{}{
            "status":  http.StatusInternalServerError,
            "message": "Failed to activate role: " + err.Error(),
            "data":    nil,
        })
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":  http.StatusOK,
        "message": "Role activated successfully",
        "data":    map[string]interface{}{"id_role": idRole},
    })
}