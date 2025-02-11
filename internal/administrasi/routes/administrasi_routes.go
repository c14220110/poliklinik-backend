package routes

import (
	"net/http"

	"github.com/c14220110/poliklinik-backend/internal/administrasi/controllers"
	"github.com/c14220110/poliklinik-backend/internal/common/middlewares"
)

func RegisterAdministrasiRoutes(ac *controllers.AdministrasiController, pc *controllers.PasienController, bc *controllers.BillingController) {
	// Login endpoint tidak dilindungi
	http.HandleFunc("/api/administrasi/login", ac.Login)

	// Registrasi pasien (misalnya, diakses via POST /api/pasien/register)
	http.Handle("/api/pasien/register", middlewares.JWTMiddlewareAdmin(http.HandlerFunc(pc.RegisterPasien)))

	// Contoh endpoint list pasien
	http.Handle("/api/administrasi/pasien", middlewares.JWTMiddlewareAdmin(http.HandlerFunc(pc.ListPasien)))

	// Endpoint billing
	http.Handle("/api/administrasi/billing", middlewares.JWTMiddlewareAdmin(http.HandlerFunc(bc.ListBilling)))
}
