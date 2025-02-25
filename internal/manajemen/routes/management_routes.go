package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
	"github.com/c14220110/poliklinik-backend/internal/manajemen/controllers"
)

func RegisterManagementLoginRoutes(mc *controllers.ManagementController) {
	// Route login tidak dilindungi oleh JWT.
	http.Handle("/api/management/login", http.HandlerFunc(mc.Login))
}

func RegisterKaryawanRoutes(kc *controllers.KaryawanController) {
	// Route untuk menambah karyawan
	http.Handle("/api/karyawan", middlewares.JWTMiddleware(http.HandlerFunc(kc.AddKaryawan)))
	// Route untuk mendapatkan daftar karyawan
	http.Handle("/api/ambilkaryawan", middlewares.JWTMiddleware(http.HandlerFunc(kc.GetKaryawanListHandler)))
	// Tambahkan route update karyawan
	http.Handle("/api/karyawan/update", middlewares.JWTMiddleware(http.HandlerFunc(kc.UpdateKaryawanHandler)))
	// Tambahkan route hapus karyawan
		http.Handle("/api/karyawan/delete", middlewares.JWTMiddleware(http.HandlerFunc(kc.SoftDeleteKaryawanHandler)))

}
