package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	"github.com/c14220110/poliklinik-backend/internal/common/middlewares" // Pastikan JWT middleware tersedia jika diperlukan
)

// RegisterAdministrasiRoutes mengatur route untuk modul administrasi.
func RegisterAdministrasiRoutes(
	adminController *controllers.AdministrasiController,
	pasienController *controllers.PasienController,
	billingController *controllers.BillingController,
) {
	// Public endpoints: Register, Login
	http.HandleFunc("/api/administrasi/register", adminController.CreateAdmin)
	http.HandleFunc("/api/administrasi/login", adminController.Login)

	// Protected endpoints (dilindungi JWT)
	http.Handle("/api/administrasi/logout", middlewares.JWTMiddleware(http.HandlerFunc(adminController.Logout)))
	http.Handle("/api/pasien/register", middlewares.JWTMiddleware(http.HandlerFunc(pasienController.RegisterPasien)))
	http.Handle("/api/pasien/list", middlewares.JWTMiddleware(http.HandlerFunc(pasienController.ListPasien)))
	http.Handle("/api/billing/list", middlewares.JWTMiddleware(http.HandlerFunc(billingController.ListBilling)))
	http.Handle("/api/billing/detail", middlewares.JWTMiddleware(http.HandlerFunc(billingController.BillingDetail)))
}
