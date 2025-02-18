package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
)

func RegisterManagementLoginRoutes(mc *controllers.ManagementController) {
	// Route login tidak dilindungi oleh middleware JWT.
	http.Handle("/api/management/login", http.HandlerFunc(mc.Login))
}

func RegisterKaryawanRoutes(kc *controllers.KaryawanController) {
	// Route untuk pengelolaan karyawan (dilindungi JWT)
	http.Handle("/api/karyawan", middlewares.JWTMiddleware(http.HandlerFunc(kc.AddKaryawan)))
	http.Handle("/api/ambilkaryawan", middlewares.JWTMiddleware(http.HandlerFunc(kc.GetKaryawanListHandler)))

}
