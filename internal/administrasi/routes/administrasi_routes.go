package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
)

func RegisterAdministrasiRoutes(ac *controllers.AdministrasiController, pc *controllers.PasienController, bc *controllers.BillingController) {
	// Login endpoint tidak dilindungi
	http.HandleFunc("/api/administrasi/login", ac.Login)

	// Endpoint lainnya dilindungi dengan JWTMiddlewareAdmin
	http.Handle("/api/administrasi/pasien", middlewares.JWTMiddlewareAdmin(http.HandlerFunc(pc.ListPasien)))
	http.Handle("/api/administrasi/billing", middlewares.JWTMiddlewareAdmin(http.HandlerFunc(bc.ListBilling)))
	// Tambahkan endpoint lain sesuai kebutuhan
}
