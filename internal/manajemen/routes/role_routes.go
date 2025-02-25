package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
)

func RegisterRoleRoutes(rc *controllers.RoleController) {
	// Route untuk menambah Role
	http.Handle("/api/role/add", middlewares.JWTMiddleware(http.HandlerFunc(rc.AddRoleHandler)))
	// Route untuk update Role
	http.Handle("/api/role/update", middlewares.JWTMiddleware(http.HandlerFunc(rc.UpdateRoleHandler)))
	// Route untuk soft delete Role
	http.Handle("/api/role/nonaktifkan", middlewares.JWTMiddleware(http.HandlerFunc(rc.SoftDeleteRoleHandler)))
	http.Handle("/api/role/aktifkan", middlewares.JWTMiddleware(http.HandlerFunc(rc.ActivateRoleHandler)))

	// Route untuk mendapatkan daftar Role
	http.Handle("/api/role/list", middlewares.JWTMiddleware(http.HandlerFunc(rc.GetRoleListHandler)))

}
